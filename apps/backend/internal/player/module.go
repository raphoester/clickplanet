package player

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_code_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/rpc_account_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_authors_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_profile_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase/publishing_record_take"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_color_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase/renaming_set_name"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_author_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/leave_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/reconcile_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_color_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/wear_title_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/forget_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/get_roster_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/move_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_out_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/stats_changed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inprocess_title_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase/notifying_award_titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/forget_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/get_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase/audit_reconcile_titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/forget_worn_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/wear_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsecrets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "player"

const (
	tileTakenBuffer      = 8192
	accountDeletedBuffer = 2048
	signInBuffer         = 256
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

	store := postgres_player_store.New(db)

	authors := get_author_usecase.New(store, players.NewGuestCodes(store, random_code_generator.Generator{}), clock)
	manyAuthors := get_authors_usecase.New(store, clock)

	accounts := rpc_account_reader.New(props.Internal)
	titleStore := postgres_title_store.New(db)
	catalog := titles.NewCatalog()
	titleBook := titles.NewBook(titleStore, catalog)
	titleFeed := inprocess_title_feed.New()
	wornTitleStore := postgres_worn_title_store.New(db)
	wardrobe := wearing.NewWardrobe(wornTitleStore, titleBook, catalog)

	visits := inmemory_visit_storage.New(clock)
	props.Runners.Add(visits)

	takes, err := cpbootstrap.Subscribe(props.Events, "player-stats", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(
			publishing_record_take.New(record_take_usecase.New(store), props.Events),
		), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	awards, err := cpbootstrap.Subscribe(props.Events, "player-titles", tileTakenBuffer,
		log_subscriber.New(stats_changed_subscriber.New(notifying_award_titles.New(
			award_titles_usecase.New(store, accounts, titleBook, clock), titleFeed,
		)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to player.v1.StatsChanged: %w", err)
	}
	deletions, err := cpbootstrap.Subscribe(props.Events, "player-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(store)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}
	forgottenTitles, err := cpbootstrap.Subscribe(props.Events, "player-titles-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_titles_usecase.New(titleStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the titles to auth.v1.AccountDeleted: %w", err)
	}
	forgottenChoices, err := cpbootstrap.Subscribe(props.Events, "player-wearing-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_worn_title_usecase.New(wornTitleStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the worn titles to auth.v1.AccountDeleted: %w", err)
	}

	forgetVisit := forget_visit_usecase.New(visits)
	signIns, err := cpbootstrap.Subscribe(props.Events, "player-presence-sign-ins", signInBuffer,
		log_subscriber.New(signed_in_subscriber.New(move_visit_usecase.New(authors, visits)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.SignedIn: %w", err)
	}
	signOuts, err := cpbootstrap.Subscribe(props.Events, "player-presence-sign-outs", signInBuffer,
		log_subscriber.New(signed_out_subscriber.New(forgetVisit), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.SignedOut: %w", err)
	}
	gone, err := cpbootstrap.Subscribe(props.Events, "player-presence-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forgetVisit), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the roster to auth.v1.AccountDeleted: %w", err)
	}
	props.Runners.Add(signOuts)
	props.Runners.Add(gone)

	props.Runners.Add(cppg.CloseAfter(db, props.Logger, takes, awards, deletions, forgottenTitles, forgottenChoices, signIns))

	verifier := rpc_session_verifier.New(props.Internal, props.Logger)

	playerService := playerv1controller.PlayerService{
		GetProfileHandler: get_profile_handler.New(get_profile_usecase.New(store)),
		SetNameHandler: set_name_handler.New(
			renaming_set_name.New(set_name_usecase.New(store, accounts, clock), visits),
		),
		SetColorHandler: set_color_handler.New(set_color_usecase.New(store)),
		GetStatsHandler: get_stats_handler.New(get_stats_usecase.New(store, clock)),
		AnnounceHandler: announce_handler.New(
			announce_usecase.New(authors, visits, cpcountries.New(), clock, tagSalt),
		),
		LeaveHandler:     leave_handler.New(forgetVisit),
		GetRosterHandler: get_roster_handler.New(get_roster_usecase.New(visits, clock)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(visits, props.Server.StreamHeartbeat),
			listen_for_titles_usecase.New(titleFeed, catalog),
		),
		GetPlayerHandler: get_player_handler.New(get_player_usecase.New(store, store, wardrobe, accounts, clock)),
		GetTitlesHandler: get_titles_handler.New(get_titles_usecase.New(store, titleBook, wardrobe, clock)),
		WearTitleHandler: wear_title_handler.New(wear_title_usecase.New(wardrobe, clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewPlayerServiceHandler(playerService, options...)
	}, playerv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
		playerv1controller.NewStreamSessionReader(verifier, clock)); err != nil {
		return fmt.Errorf("failed to mount player.v1.PlayerService: %w", err)
	}

	internalService := playerv1controller.InternalService{
		GetAuthorHandler:  get_author_handler.New(authors),
		GetAuthorsHandler: get_authors_handler.New(manyAuthors),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount player.v1.InternalService: %w", err)
	}

	adminService := playerv1controller.AdminService{
		ReconcileTitlesHandler: reconcile_titles_handler.New(audit_reconcile_titles.New(
			reconcile_titles_usecase.New(store, accounts, titleStore, catalog, clock), props.Logger,
		)),
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

	TagSalt string
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("player.database: %w", err)
	}
	return nil
}
