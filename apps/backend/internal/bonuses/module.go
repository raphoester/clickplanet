package bonuses

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/box_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/rpc_planet_wallet"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/answer_quiz_usecase/prom_answer_quiz"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/claim_bonus_usecase/prom_claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/grant_sign_in_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/answer_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_grant_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query/inmemory_quiz_questions"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/open_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/rpc_planet_freeze"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes/rpc_planet_shares"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings/postgres_rating_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings/usecases/record_answer_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings/usecases/record_answer_usecase/log_record_answer"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings/usecases/record_answer_usecase/prom_record_answer"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/click_tried_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/player_created_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName = "bonuses"

	factsRelayInterval = 100 * time.Millisecond

	// How often the quiz's lean asks planet what each flag holds: a lean a few seconds old leans the same way.
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
		return fmt.Errorf("the bonuses module calls planet: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the bonuses to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the bonuses to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	ruleParts := parts.All()
	ruleSets := cprules.NewPostgresStore(db, ruleParts)
	inForce := cprules.NewInForce[*bonusesv1.Rules](clock)
	playRules := cprules.NewLoggedPlay(cprules.NewPlayUseCase(ruleSets, cprules.NewPostgresChoices(db), inForce, nil),
		props.Logger, cprules.PlayLog{Taken: "offer rules in force", Failed: "failed to play the rules"})
	if _, _, err := playRules.Execute(ctx); err != nil {
		closeStores()
		return fmt.Errorf("failed to play the offers' rules postgres keeps: %w", err)
	}
	offerRules := cprules.Reader(inForce, bonuses.Part)

	// The ratings answers moved, read once: the draw reads them in memory, over each question's prior.
	ratingStore := postgres_rating_store.New(db)
	rated, err := ratingStore.Ratings(ctx)
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to read the quiz questions' ratings: %w", err)
	}
	book := ratings.NewBook(rated)

	shares := rpc_planet_shares.New(planet, sharesInterval)
	bank, err := quizzes.Load(shares, book)
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to load the quiz bank: %w", err)
	}
	props.Logger.Info("quiz bank loaded",
		slog.String("bank", bank.Version()),
		slog.Int("questions", bank.Size()),
		slog.Int("subjects", bank.Subjects()),
		slog.Int("rated", len(rated)))
	answers := log_record_answer.New(prom_record_answer.New(
		record_answer_usecase.New(db, ratingStore, bank, book), props.Metrics), props.Logger)

	// An offer's payment is named by its token, which planet knows in memory as the offer is; the refill's, for good.
	offersWallet := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile)
	signInWallet := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForGood)

	registry := bonuses.New(config.Offers, clock, offersWallet, offerRules, bank)
	told := facts.NewLog(clock)
	taken := take(registry, offersWallet, answers, streams, told, clock, props)

	clicks, err := cpredis.Subscribe(ctx, streams, "bonuses-clicks",
		log_subscriber.New(click_tried_subscriber.New(registry), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the offers to planet.v1.ClickTried: %w", err)
	}
	// Planet's group name, kept: the group goes on where planet's subscriber left the stream.
	newPlayers, err := cpredis.Subscribe(ctx, streams, "planet-charges-players",
		log_subscriber.New(player_created_subscriber.New(
			grant_sign_in_refill_usecase.New(signInWallet, offerRules)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the sign-in refill to player.v1.PlayerCreated: %w", err)
	}
	forgotten, err := cpredis.Subscribe(ctx, streams, "bonuses-answers-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_account_usecase.New(ratingStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the quiz answers to player.v1.PlayerForgotten: %w", err)
	}

	props.Runners.Add(registry)
	props.Runners.Add(shares)
	// Not closers: the subscribers and the relay's last send outlive the server.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		clicks, newPlayers, forgotten, facts.NewRelay(told, streams, factsRelayInterval))))

	freeze := rpc_planet_freeze.New(planet)
	service := bonusesv1controller.BonusService{
		ClaimBonusHandler: claim_bonus_handler.New(taken.claim),
		OpenQuizHandler:   open_quiz_handler.New(open_quiz_usecase.New(registry)),
		AnswerQuizHandler: answer_quiz_handler.New(taken.answer),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(registry, inForce, props.Server.StreamHeartbeat)),
		GetGrantRulesHandler: get_grant_rules_handler.New(offerRules),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return bonusesv1connect.NewBonusServiceHandler(service, options...)
	},
		bonusesv1controller.NewFreezeInterceptor(func(ctx context.Context) (bool, error) {
			if offerRules().Frozen {
				return true, nil
			}
			return freeze.Frozen(ctx)
		}),
		bonusesv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics),
		bonusesv1controller.NewStreamSessionReader(props.Sessions, clock),
	); err != nil {
		return fmt.Errorf("failed to mount bonuses.v1.BonusService: %w", err)
	}

	admin := bonusesv1controller.AdminService{
		GetRulesHandler: get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
		SaveRuleSetHandler: save_rule_set_handler.New(
			cprules.NewAuditedSave(cprules.NewSaveUseCase(db, ruleSets, ruleParts, playRules), props.Logger,
				cprules.SaveLog{Saved: "admin offer rule set saved", Refused: "admin offer rule set refused"})),
		ListQuizQuestionsHandler: list_quiz_questions_handler.New(
			quiz_questions_query.NewMemoryQuery(inmemory_quiz_questions.New(bank))),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return bonusesv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount bonuses.v1.AdminService: %w", err)
	}

	internalService := bonusesv1controller.InternalService{
		PlayRuleSetHandler:     play_rule_set_handler.New(playRules),
		GetRuleSetNamesHandler: get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
		GetRuleSetHandler:      get_rule_set_handler.New(rule_set_query.NewPostgresQuery(db)),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return bonusesv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount bonuses.v1.InternalService: %w", err)
	}

	return nil
}

// taking is the claim and the answer as the module counts them: every metric keeps the name it had in planet.
type taking struct {
	claim  *prom_claim_bonus.Decorator
	answer *prom_answer_quiz.Decorator
}

// take builds the claim and the answer, counts the offers, and tells the antibot each box caught, lapsed or claimed
// by another scope.
func take(
	registry *bonuses.Registry, wallet rpc_planet_wallet.Wallet, answers answer_quiz_usecase.Answers,
	streams *cpredis.Streams, told *facts.Log, clock cptime.Clock, props cpbootstrap.Props,
) taking {
	claim, counters := prom_claim_bonus.New(claim_bonus_usecase.New(registry, wallet, streams, clock), props.Metrics)
	answer, quizCounters := prom_answer_quiz.New(
		answer_quiz_usecase.New(registry, wallet, answers, streams, clock), props.Metrics)

	boxes := box_facts.New(told)
	registry.Observe(bonuses.Report{
		Offered: counters.Offered.Inc,
		Lapsed: func(scope string) {
			counters.Lapsed.Inc()
			boxes.Lapsed(scope)
		},
		Caught: func(scope string, after time.Duration) {
			counters.Caught.Observe(after.Seconds())
			boxes.Caught(scope, after)
		},
		Foreign: func(scope string) {
			counters.Foreign.Inc()
			boxes.Foreign(scope)
		},

		QuizOffered: quizCounters.Offered.Inc,
		QuizLapsed:  func(string) { quizCounters.Lapsed.Inc() },
		QuizAnswered: func(_ string, correct bool, after time.Duration) {
			quizCounters.Answered.WithLabelValues(strconv.FormatBool(correct)).Observe(after.Seconds())
		},
	})

	return taking{claim: claim, answer: answer}
}
