package quizzes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank/rpc_planet_shares"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/answer_quiz_usecase/prom_answer_quiz"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/answer_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/list_quiz_questions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/list_quiz_questions_handler/quiz_questions_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/list_quiz_questions_handler/quiz_questions_query/inmemory_quiz_questions"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/open_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/rpc_planet_freeze"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/postgres_rating_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/record_answer_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/record_answer_usecase/log_record_answer"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/record_answer_usecase/prom_record_answer"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/subscribers/click_tried_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName = "quizzes"

	// How often the lean asks planet what each flag holds: a lean a few seconds old leans the same way.
	sharesInterval = 10 * time.Second
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
		return fmt.Errorf("the quizzes module calls planet: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the quizzes to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the quizzes to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	ruleParts := parts.All()
	ruleSets := cprules.NewPostgresStore(db, ruleParts)
	inForce := cprules.NewInForce[*quizzesv1.Rules](clock)
	playRules := cprules.NewLoggedPlay(cprules.NewPlayUseCase(ruleSets, cprules.NewPostgresChoices(db), inForce, nil),
		props.Logger, cprules.PlayLog{Taken: "quiz rules in force", Failed: "failed to play the quiz rules"})
	if _, _, err := playRules.Execute(ctx); err != nil {
		closeStores()
		return fmt.Errorf("failed to play the quizzes' rules postgres keeps: %w", err)
	}
	quizRules := cprules.Reader(inForce, quizzes.Part)

	// The ratings answers moved, read once: the draw reads them in memory, over each question's prior.
	ratingStore := postgres_rating_store.New(db)
	rated, err := ratingStore.Ratings(ctx)
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to read the quiz questions' ratings: %w", err)
	}
	book := ratings.NewBook(rated)

	shares := rpc_planet_shares.New(planet, sharesInterval)
	questions, err := bank.Load(shares, book)
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to load the quiz bank: %w", err)
	}
	props.Logger.Info("quiz bank loaded",
		slog.String("bank", questions.Version()),
		slog.Int("questions", questions.Size()),
		slog.Int("subjects", questions.Subjects()),
		slog.Int("rated", len(rated)))
	answers := log_record_answer.New(prom_record_answer.New(
		record_answer_usecase.New(db, ratingStore, questions, book), props.Metrics), props.Logger)

	registry := quizzes.New(config.Offers, clock, quizRules, questions)
	answer, counters := prom_answer_quiz.New(
		answer_quiz_usecase.New(registry, answers, db, cpoutbox.New(db), clock), props.Metrics)
	registry.Observe(quizzes.Report{
		Offered: counters.Offered.Inc,
		Lapsed:  func(string) { counters.Lapsed.Inc() },
		Answered: func(_ string, correct bool, after time.Duration) {
			counters.Answered.WithLabelValues(strconv.FormatBool(correct)).Observe(after.Seconds())
		},
	})

	clicks, err := cpredis.Subscribe(ctx, streams, "quizzes-clicks",
		log_subscriber.New(click_tried_subscriber.New(registry), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the quizzes to planet.v1.ClickTried: %w", err)
	}
	// The bonuses module's group name, kept: the group goes on where the bonuses module left the stream.
	forgotten, err := cpredis.Subscribe(ctx, streams, "bonuses-answers-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_account_usecase.New(ratingStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the quiz answers to player.v1.PlayerForgotten: %w", err)
	}

	props.Runners.Add(registry)
	props.Runners.Add(shares)
	// Not closers: the subscribers and the outbox's last send outlive the server.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		clicks, forgotten, cpoutbox.NewRelay("quizzes-outbox", db, streams, props.Logger))))

	freeze := rpc_planet_freeze.New(planet)
	service := quizzesv1controller.QuizService{
		OpenQuizHandler:   open_quiz_handler.New(open_quiz_usecase.New(registry)),
		AnswerQuizHandler: answer_quiz_handler.New(answer),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(registry, props.Server.StreamHeartbeat)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return quizzesv1connect.NewQuizServiceHandler(service, options...)
	},
		quizzesv1controller.NewFreezeInterceptor(func(ctx context.Context) (bool, error) {
			if quizRules().Frozen {
				return true, nil
			}
			return freeze.Frozen(ctx)
		}),
		quizzesv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics),
		quizzesv1controller.NewStreamSessionReader(props.Sessions, clock),
	); err != nil {
		return fmt.Errorf("failed to mount quizzes.v1.QuizService: %w", err)
	}

	admin := quizzesv1controller.AdminService{
		GetRulesHandler: get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
		SaveRuleSetHandler: save_rule_set_handler.New(
			cprules.NewAuditedSave(cprules.NewSaveUseCase(db, ruleSets, ruleParts, playRules), props.Logger,
				cprules.SaveLog{Saved: "admin quiz rule set saved", Refused: "admin quiz rule set refused"})),
		ListQuizQuestionsHandler: list_quiz_questions_handler.New(
			quiz_questions_query.NewMemoryQuery(inmemory_quiz_questions.New(questions))),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return quizzesv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount quizzes.v1.AdminService: %w", err)
	}

	internalService := quizzesv1controller.InternalService{
		PlayRuleSetHandler:     play_rule_set_handler.New(playRules),
		GetRuleSetNamesHandler: get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
		GetRuleSetHandler:      get_rule_set_handler.New(rule_set_query.NewPostgresQuery(db)),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return quizzesv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount quizzes.v1.InternalService: %w", err)
	}

	return nil
}
