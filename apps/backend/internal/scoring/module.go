package scoring

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/rpc_planet_territories"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/count_take_usecase"
	round_forget_account_usecase "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/log_command"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/scoring_command_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
	board_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler/my_tally_query"
	tally_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler/my_tally_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler/round_query"
	round_authors "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler/round_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler/scores_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_table_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/log_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/round_take_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName = "scoring"

	boardsEvery = 2 * time.Second
	scoresEvery = 10 * time.Second
	roundsEvery = 10 * time.Second
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
	countries := cpcountries.New()

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the scoring module asks the player module who plays: %w", err)
	}
	player := playerv1connect.NewInternalServiceClient(internal, baseURL)
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the scoring module to postgres: %w", err)
	}
	if err := db.RenameSchemaFrom(ctx, migrations.FormerSchema); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to rename the %s schema: %w", migrations.FormerSchema, err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the scoring module to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	bookStore := postgres_book_store.New(db)
	roundStore := postgres_round_store.New(db)
	contributions := postgres_contribution_store.New(db)
	territories := rpc_planet_territories.New(planet)
	outbox := cpoutbox.New(db)

	// The groups keep their names: a new name is a new group, which reads TileTaken again from the start of the stream.
	takes, err := cpredis.Subscribe(ctx, streams, "seasons-standings",
		log_subscriber.New(tile_taken_subscriber.New(record_take_usecase.New(bookStore, contributions)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to planet.v1.TileTaken: %w", err)
	}
	deletions, err := cpredis.Subscribe(ctx, streams, "seasons-standings-accounts",
		log_subscriber.New(account_deleted_subscriber.New(forget_account_usecase.New(contributions)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}
	roundTakes, err := cpredis.Subscribe(ctx, streams, "seasons-rounds",
		log_subscriber.New(round_take_subscriber.New(count_take_usecase.New(bookStore, roundStore, roundStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the rounds to planet.v1.TileTaken: %w", err)
	}
	roundDeletions, err := cpredis.Subscribe(ctx, streams, "seasons-rounds-accounts",
		log_subscriber.New(account_deleted_subscriber.New(round_forget_account_usecase.New(roundStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the rounds to auth.v1.AccountDeleted: %w", err)
	}

	commands, err := cpredis.Handle(ctx, streams, scoring_command_handler.New(log_subscriber.New[*scoringv1.ScoringCommand](
		scoring_command_handler.NewDispatcher(scoring_command_handler.UseCases{
			OpenBook:   log_command.New("open_book", open_book_usecase.New(bookStore), props.Logger),
			OpenRound:  log_command.New("open_round", open_round_usecase.New(bookStore, roundStore), props.Logger),
			CloseRound: log_command.New("close_round", close_round_usecase.New(db, territories, roundStore, outbox), props.Logger),
			CloseBook:  log_command.New("close_book", close_book_usecase.New(db, bookStore, roundStore, outbox), props.Logger),
		}), props.Logger)))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to handle scoring.v1.ScoringCommand: %w", err)
	}

	boardQuery := board_query.NewPostgresQuery(db, board_authors.New(player), countries)
	scoresQuery := scores_query.NewPostgresQuery(db)
	roundQuery := round_query.NewPostgresQuery(db, territories, round_authors.New(player), scoresQuery)
	boards := inprocess_feed.New[board_query.View, *scoringv1.Board]("scoring-boards",
		log_reader.New[board_query.View, *scoringv1.Board]("board", boardQuery, props.Logger), clock, boardsEvery)
	scores := inprocess_feed.New[books.BookID, *scoringv1.Scores]("scoring-scores",
		log_reader.New[books.BookID, *scoringv1.Scores]("scores", scoresQuery, props.Logger), clock, scoresEvery)
	rounds := inprocess_feed.New[books.RoundID, *scoringv1.Round]("scoring-rounds",
		log_reader.New[books.RoundID, *scoringv1.Round]("round", roundQuery, props.Logger), clock, roundsEvery)

	props.Runners.Add(cppg.CloseAfter(db, props.Logger, boards, scores, rounds, cpredis.CloseAfter(streams, props.Logger,
		takes, deletions, roundTakes, roundDeletions, commands, cpoutbox.NewRelay("scoring-outbox", db, streams, props.Logger))))

	getRound := get_round_handler.New(roundQuery)
	service := scoringv1controller.ScoringService{
		GetBoardHandler:   get_board_handler.New(boardQuery),
		GetMyTallyHandler: get_my_tally_handler.New(my_tally_query.NewPostgresQuery(db, tally_authors.New(player), countries)),
		GetRoundHandler:   getRound,
		GetScoresHandler:  get_scores_handler.New(scoresQuery),
		ListenForEventsHandler: listen_for_events_handler.New(listen_for_events_handler.Feeds{
			Boards: boards, Scores: scores, Rounds: rounds,
		}, countries, props.Server.StreamHeartbeat),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return scoringv1connect.NewScoringServiceHandler(service, options...)
	}, scoringv1controller.NewCacheInterceptor(),
		scoringv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
	); err != nil {
		return fmt.Errorf("failed to mount scoring.v1.ScoringService: %w", err)
	}

	internalService := scoringv1controller.InternalService{
		GetRoundHandler: getRound,
		GetTableHandler: get_table_handler.New(scoresQuery),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return scoringv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount scoring.v1.InternalService: %w", err)
	}

	props.Logger.Info("scoring built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("scoring.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("scoring.redis: %w", err)
	}
	return nil
}
