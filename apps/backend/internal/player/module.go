// Package player wires what the game keeps about one player: the name it chose and its stats.
//
// It makes no account and verifies no token of its own. The account is the one the click token names,
// checked with the key auth hands over the internal listener; the stats come from planet.v1.TileTaken, and
// auth.v1.AccountDeleted forgets both. It imports neither module: only their proto packages.
package player

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage/log_flush"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_profile_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config, props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	clock := cptime.SystemClock{}

	// ---- Storage ----

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the player module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	storage := inmemory_player_storage.New(postgres_player_store.New(db))
	if err := storage.Load(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to load the players: %w", err)
	}

	// ---- Events ----

	// Subscribed here, before any runner starts, so a take published at boot waits in the buffer.
	takes, err := cpbootstrap.Subscribe(props.Events, "player-stats", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(record_take_usecase.New(storage)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpbootstrap.Subscribe(props.Events, "player-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(storage)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	// Stopped in order: the subscribers drain what is buffered, then the last flush writes it, then the pool closes.
	flusher := inmemory_player_storage.NewRunner(config.Storage, log_flush.New(storage, props.Logger))
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpbootstrap.Pipeline(takes, deletions, flusher)))

	// ---- Player service ----

	// The key comes from auth over the internal listener, on the first call: this module holds no seed.
	verifier := rpc_session_verifier.New(props.Internal, props.Logger)

	playerService := playerv1controller.PlayerService{
		GetProfileHandler: get_profile_handler.New(get_profile_usecase.New(storage)),
		SetNameHandler:    set_name_handler.New(set_name_usecase.New(storage, clock)),
		GetStatsHandler:   get_stats_handler.New(get_stats_usecase.New(storage, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewPlayerServiceHandler(playerService, options...)
	}, playerv1controller.NewSessionInterceptor(verifier, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount player.v1.PlayerService: %w", err)
	}

	// ---- Internal service ----

	internalService := playerv1controller.InternalService{
		GetNamesHandler: get_names_handler.New(get_names_usecase.New(storage)),
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
	// Off registers nothing: player.v1 404s and nobody hears the events.
	Enabled bool

	// Profiles and stats, in their own schema. Required when the module is on.
	Database cppg.Config

	Storage inmemory_player_storage.Config
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("player.database: %w", err)
	}
	return nil
}
