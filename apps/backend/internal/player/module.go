// Package player wires what the game keeps about one player: the name it chose, its tag and its stats.
//
// It makes no account and verifies no token of its own. The account is the one the click token names,
// checked with the key auth hands over the internal listener, and auth is asked there too whether it may hold
// a username; the stats come from planet.v1.TileTaken, and auth.v1.AccountDeleted forgets both. It imports neither module: only their proto packages.
package player

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/rpc_account_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_profile_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_author_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/get_roster_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsecrets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "player"

const (
	// A spread takes up to 7 tiles a click; a take is one map write, so the buffer only fills if the process stalls.
	tileTakenBuffer      = 8192
	accountDeletedBuffer = 2048
)

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	clock := cptime.SystemClock{}

	tagSalt := config.TagSalt
	if tagSalt == "" {
		salt, err := cpsecrets.RandomHex()
		if err != nil {
			return fmt.Errorf("failed to generate a tag salt: %w", err)
		}
		tagSalt = salt
		props.Logger.Warn("no player.tagSalt configured, generated a random one: every tag will change on every restart")
	}

	// ---- Storage ----

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the player module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	// Every call reads or writes postgres: a click never waits on it, since takes arrive over the event bus.
	store := postgres_player_store.New(db)

	// ---- Events ----

	// Subscribed here, before any runner starts, so a take published at boot waits in the buffer.
	// A full buffer drops a take and counts it; a slow database fills it, never a click.
	takes, err := cpbootstrap.Subscribe(props.Events, "player-stats", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(record_take_usecase.New(store)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpbootstrap.Subscribe(props.Events, "player-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(store)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	// The pool closes once both subscribers have drained what is buffered: closers run before the runners stop.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, takes, deletions))

	// ---- Presence ----

	// Who is playing, in memory only: a restart empties it and the clients fill it again within 30s.
	visits := inmemory_visit_storage.New(clock)
	props.Runners.Add(visits)

	// ---- Player service ----

	// The key comes from auth over the internal listener, on the first call: this module holds no seed.
	verifier := rpc_session_verifier.New(props.Internal, props.Logger)
	accounts := rpc_account_reader.New(props.Internal)

	playerService := playerv1controller.PlayerService{
		GetProfileHandler: get_profile_handler.New(get_profile_usecase.New(store)),
		// Only a linked account may hold a username, and auth is asked on each SetName.
		SetNameHandler:  set_name_handler.New(set_name_usecase.New(store, accounts, clock)),
		GetStatsHandler: get_stats_handler.New(get_stats_usecase.New(store, clock)),
		AnnounceHandler: announce_handler.New(
			announce_usecase.New(store, visits, cpcountries.New(), clock, tagSalt)),
		GetRosterHandler: get_roster_handler.New(get_roster_usecase.New(visits, clock)),
		// Anybody may open a player: auth is asked when its account was made, on each call.
		GetPlayerHandler: get_player_handler.New(get_player_usecase.New(store, store, accounts, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewPlayerServiceHandler(playerService, options...)
	}, playerv1controller.NewSessionInterceptor(verifier, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount player.v1.PlayerService: %w", err)
	}

	// ---- Internal service ----

	internalService := playerv1controller.InternalService{
		GetAuthorHandler: get_author_handler.New(get_author_usecase.New(store, tagSalt)),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount player.v1.InternalService: %w", err)
	}

	props.Logger.Info("player built", slog.String("schema", config.Database.Schema))

	return nil
}

// Config is the `player:` block.
type Config struct {
	// Profiles and stats, in their own schema. Required: the module is always on, since the chat asks it who
	// posts.
	Database cppg.Config

	// TagSalt salts the hash of an address that the game shows beside every name. Left empty, one is generated
	// at boot, and every tag changes on each restart.
	TagSalt string
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("player.database: %w", err)
	}
	return nil
}
