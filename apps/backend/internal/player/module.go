package player

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/postgres_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/usecases/forget_fronts_usecase"
	record_front_take "github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_code_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_name_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/rpc_account_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase/renaming_name_account"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_accounts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_accounts_usecase/audit_name_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase/publishing_record_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase/publishing_record_take"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_color_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase/renaming_set_name"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_author_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler/authors_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_fronts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_fronts_handler/fronts_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler/profile_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query/inmemory_roster"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler/stats_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler/titles_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/leave_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/name_accounts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/reconcile_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_color_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/wear_title_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/forget_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/move_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/message_sent_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_account_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_out_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/stats_changed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_fronts_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inprocess_title_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase/notifying_award_titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/forget_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase/audit_reconcile_titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/forget_worn_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/wear_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/wear_title_usecase/dressing_wear_title"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsecrets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "player"

const (
	tileTakenBuffer      = 8192
	accountDeletedBuffer = 2048
	signInBuffer         = 256
	messageSentBuffer    = 256
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

	store := postgres_player_store.New(db)

	accounts := rpc_account_reader.New(auth)
	titleStore := postgres_title_store.New(db)
	catalog := titles.NewCatalog()
	titleBook := titles.NewBook(titleStore, catalog)
	titleFeed := inprocess_title_feed.New()
	wornTitleStore := postgres_worn_title_store.New(db)
	frontStore := postgres_front_store.New(db)
	wardrobe := wearing.NewWardrobe(wornTitleStore, titleBook, catalog)
	titleCards := inprocess_title_catalog.New(catalog)

	authors := get_author_usecase.New(store, players.NewGuestCodes(store, random_code_generator.Generator{}), wardrobe, clock)

	visits := inmemory_visit_storage.New(clock)
	generatedNames := players.NewGeneratedNames(store, random_name_generator.Generator{})
	props.Runners.Add(visits)

	takes, err := cpbootstrap.Subscribe(props.Events, "player-stats", tileTakenBuffer,
		log_subscriber.New(tile_taken_subscriber.New(
			publishing_record_take.New(record_take_usecase.New(store), props.Events),
		), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	frontTakes, err := cpbootstrap.Subscribe(props.Events, "player-fronts", tileTakenBuffer,
		log_subscriber.New(tile_taken_fronts_subscriber.New(record_front_take.New(frontStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the fronts to planet.v1.TileTaken: %w", err)
	}
	posts, err := cpbootstrap.Subscribe(props.Events, "player-stats-messages", messageSentBuffer,
		log_subscriber.New(message_sent_subscriber.New(
			publishing_record_message.New(record_message_usecase.New(store), props.Events),
		), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to chat.v1.MessageSent: %w", err)
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

	forgottenFronts, err := cpbootstrap.Subscribe(props.Events, "player-fronts-accounts", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_fronts_usecase.New(frontStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the fronts to auth.v1.AccountDeleted: %w", err)
	}

	forgetVisit := forget_visit_usecase.New(visits)
	signIns, err := cpbootstrap.Subscribe(props.Events, "player-presence-sign-ins", signInBuffer,
		log_subscriber.New(signed_in_subscriber.New(move_visit_usecase.New(authors, visits)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.SignedIn: %w", err)
	}
	namings, err := cpbootstrap.Subscribe(props.Events, "player-names-sign-ins", signInBuffer,
		log_subscriber.New(signed_in_account_subscriber.New(renaming_name_account.New(
			name_account_usecase.New(generatedNames, store, clock), visits,
		)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the names to auth.v1.SignedIn: %w", err)
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

	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		takes, frontTakes, posts, awards, deletions, forgottenTitles, forgottenChoices, forgottenFronts, signIns, namings))

	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", "player")))

	playerService := playerv1controller.PlayerService{
		GetProfileHandler: get_profile_handler.New(profile_query.NewPostgresQuery(db)),
		SetNameHandler: set_name_handler.New(
			renaming_set_name.New(set_name_usecase.New(store, accounts, clock), visits),
		),
		SetColorHandler: set_color_handler.New(set_color_usecase.New(store)),
		GetStatsHandler: get_stats_handler.New(stats_query.NewPostgresQuery(db, clock)),
		AnnounceHandler: announce_handler.New(
			announce_usecase.New(authors, visits, cpcountries.New(), clock, tagSalt),
		),
		LeaveHandler:     leave_handler.New(forgetVisit),
		GetRosterHandler: get_roster_handler.New(roster_query.NewMemoryQuery(inmemory_roster.New(visits, clock))),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(visits, props.Server.StreamHeartbeat),
			listen_for_titles_usecase.New(titleFeed, catalog),
		),
		GetPlayerHandler: get_player_handler.New(player_query.NewPostgresQuery(db, titleCards, accounts, clock)),
		GetTitlesHandler: get_titles_handler.New(titles_query.NewPostgresQuery(db, titleCards, clock)),
		WearTitleHandler: wear_title_handler.New(dressing_wear_title.New(wear_title_usecase.New(wardrobe, clock), visits)),
		GetFrontsHandler: get_fronts_handler.New(fronts_query.NewPostgresQuery(db)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return playerv1connect.NewPlayerServiceHandler(playerService, options...)
	}, playerv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
		playerv1controller.NewStreamSessionReader(verifier, clock)); err != nil {
		return fmt.Errorf("failed to mount player.v1.PlayerService: %w", err)
	}

	internalService := playerv1controller.InternalService{
		GetAuthorHandler:  get_author_handler.New(authors),
		GetAuthorsHandler: get_authors_handler.New(authors_query.NewPostgresQuery(db, titleCards, clock)),
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
		NameAccountsHandler: name_accounts_handler.New(audit_name_accounts.New(
			name_accounts_usecase.New(store, accounts, generatedNames, clock), props.Logger,
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
