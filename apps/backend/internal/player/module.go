package player

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/usecases/decide_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/usecases/forget_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_code_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_name_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/rpc_account_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/introduce_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase/renaming_name_account"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase/audit_rename_player"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_color_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase/renaming_set_name"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/tell_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler/admin_player_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_accounts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_nameplates_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_players_handler/players_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler/profile_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query/inmemory_roster"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_rename_rule"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/introduce_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/leave_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rename_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_handler/matches_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_color_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/forget_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/move_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/deletion_requested_account_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/deletion_requested_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/restored_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_account_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_out_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsecrets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "player"

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

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the player module asks auth about accounts: %w", err)
	}
	auth := authv1connect.NewInternalServiceClient(internal, baseURL)

	tagSalt := config.TagSalt
	if tagSalt == "" {
		salt, err := cpsecrets.RandomHex()
		if err != nil {
			return fmt.Errorf("failed to generate a tag salt: %w", err)
		}
		tagSalt = salt
		props.Logger.Info("no player.tagSalt configured, generated a random one")
	}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the player module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the player module to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	store := postgres_player_store.New(db)
	outbox := cpoutbox.New(db)

	accounts := rpc_account_reader.New(auth)
	departureStore := postgres_departure_store.New(db)

	introducer := introduce_player_usecase.New(store, players.NewGuestCodes(store, random_code_generator.Generator{}))

	visits := inmemory_visit_storage.New(clock)
	generatedNames := players.NewGeneratedNames(store, random_name_generator.Generator{})
	props.Runners.Add(visits)

	// Not player-accounts any more: a new group reads every AccountDeleted still in the stream, and tells each one
	// forgotten again, for the modules that heard AccountDeleted until 2026-11-04 (see docs/player.md).
	deletions, err := cpredis.Subscribe(ctx, streams, "player-profiles-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(db, store, outbox)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	decideDeparture := decide_departure_usecase.New(db, departureStore, outbox)
	requests, err := cpredis.Subscribe(ctx, streams, "player-departures-requests",
		log_subscriber.New(deletion_requested_subscriber.New(decideDeparture), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeletionRequested: %w", err)
	}
	restores, err := cpredis.Subscribe(ctx, streams, "player-departures-restores",
		log_subscriber.New(restored_subscriber.New(decideDeparture), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountRestored: %w", err)
	}
	forgottenDepartures, err := cpredis.Subscribe(ctx, streams, "player-departures-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_departure_usecase.New(departureStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the departures to auth.v1.AccountDeleted: %w", err)
	}

	forgetVisit := forget_visit_usecase.New(visits)
	signIns, err := cpredis.Subscribe(ctx, streams, "player-presence-sign-ins",
		log_subscriber.New(signed_in_subscriber.New(move_visit_usecase.New(introducer, visits)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.SignedIn: %w", err)
	}
	namings, err := cpredis.Subscribe(ctx, streams, "player-names-sign-ins",
		log_subscriber.New(signed_in_account_subscriber.New(renaming_name_account.New(
			name_account_usecase.New(db, generatedNames, store, outbox, clock), visits,
		)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the names to auth.v1.SignedIn: %w", err)
	}
	told, err := cpredis.Subscribe(ctx, streams, "player-sign-ins",
		log_subscriber.New(signed_in_told_subscriber.New(tell_sign_in_usecase.New(db, outbox)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the sign-ins told to auth.v1.SignedIn: %w", err)
	}
	signOuts, err := cpredis.Subscribe(ctx, streams, "player-presence-sign-outs",
		log_subscriber.New(signed_out_subscriber.New(forgetVisit), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.SignedOut: %w", err)
	}
	gone, err := cpredis.Subscribe(ctx, streams, "player-presence-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forgetVisit), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the roster to auth.v1.AccountDeleted: %w", err)
	}
	leaving, err := cpredis.Subscribe(ctx, streams, "player-presence-deletions",
		log_subscriber.New(deletion_requested_account_subscriber.New(forgetVisit), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the roster to auth.v1.AccountDeletionRequested: %w", err)
	}
	// Not closers: closers run before the runners stop, and the runners use both.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		deletions, requests, restores, forgottenDepartures, signIns, namings, told, signOuts, gone, leaving,
		cpoutbox.NewRelay("player-outbox", db, streams, props.Logger))))

	getRoster := get_roster_handler.New(roster_query.NewMemoryQuery(inmemory_roster.New(visits, clock)))
	playersQuery := players_query.NewPostgresQuery(db)
	playerService := playerv1controller.PlayerService{
		GetProfileHandler: get_profile_handler.New(profile_query.NewPostgresQuery(db, inprocess_rename_rule.New())),
		SetNameHandler: set_name_handler.New(
			renaming_set_name.New(set_name_usecase.New(db, store, accounts, outbox, clock), visits),
		),
		SetColorHandler: set_color_handler.New(set_color_usecase.New(store)),
		AnnounceHandler: announce_handler.New(
			announce_usecase.New(introducer, visits, cpcountries.New(), clock, tagSalt),
		),
		LeaveHandler:     leave_handler.New(forgetVisit),
		GetRosterHandler: getRoster,
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(visits, props.Server.StreamHeartbeat),
		),
		GetPlayerHandler:     get_player_handler.New(player_query.NewPostgresQuery(db, accounts)),
		GetNameplatesHandler: get_nameplates_handler.New(playersQuery),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewPlayerServiceHandler(playerService, options...)
	}, playerv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics)); err != nil {
		return fmt.Errorf("failed to mount player.v1.PlayerService: %w", err)
	}

	getPlayers := get_players_handler.New(playersQuery)
	internalService := playerv1controller.InternalService{
		IntroducePlayerHandler: introduce_player_handler.New(introducer),
		GetPlayersHandler:      getPlayers,
		GetAccountsHandler:     get_accounts_handler.New(accounts),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount player.v1.InternalService: %w", err)
	}

	adminService := playerv1controller.AdminService{
		GetPlayersHandler:     getPlayers,
		AdminGetPlayerHandler: admin_get_player_handler.New(admin_player_query.NewPostgresQuery(db, accounts)),
		GetRosterHandler:      getRoster,
		SearchPlayersHandler:  search_players_handler.New(matches_query.NewPostgresQuery(db)),
		RenamePlayerHandler: rename_player_handler.New(
			audit_rename_player.New(rename_player_usecase.New(db, store, outbox, clock), props.Logger),
		),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount player.v1.AdminService: %w", err)
	}

	props.Logger.Info("player built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config

	TagSalt string
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("player.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("player.redis: %w", err)
	}
	return nil
}
