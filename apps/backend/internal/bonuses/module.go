package bonuses

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/rpc_planet_wallet"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/grant_sign_in_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase/prom_pay_caught_box"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_level_up_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_right_answer_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_grant_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/commands/keep_unlocks_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/commands/pay_level_up_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/box_claimed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/player_created_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/quiz_answered_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/postgres_unlock_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/usecases/forget_unlocks_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/usecases/keep_unlocks_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "bonuses"

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
		return fmt.Errorf("failed to play the bonuses' rules postgres keeps: %w", err)
	}
	winRules := cprules.Reader(inForce, bonuses.Part)

	// A box's or a right answer's payment is named by its token, which planet knows in memory; the refill's and a
	// level-up's, for good.
	winsWallet := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForAWhile)
	lastingWallet := rpc_planet_wallet.New(planet, rpc_planet_wallet.ForGood)
	unlockStore := postgres_unlock_store.New(db)

	// Planet's group name, kept: the group goes on where planet's subscriber left the stream.
	newPlayers, err := cpredis.Subscribe(ctx, streams, "planet-charges-players",
		log_subscriber.New(player_created_subscriber.New(
			grant_sign_in_refill_usecase.New(lastingWallet, winRules)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the sign-in refill to player.v1.PlayerCreated: %w", err)
	}
	// Only the quizzes' own stream: an answer the bonuses module told was paid as it was answered.
	rightAnswers, err := cpredis.Subscribe(ctx, streams, "bonuses-right-answers",
		log_subscriber.New(quiz_answered_subscriber.New(
			pay_right_answer_usecase.New(winsWallet, unlockStore, winRules)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the rewards to quizzes.v1.QuizAnswered: %w", err)
	}
	// Only the boxes' own stream: a box the bonuses module offered was paid as it was caught.
	boxesCaught, err := cpredis.Subscribe(ctx, streams, "bonuses-boxes",
		log_subscriber.New(box_claimed_subscriber.New(prom_pay_caught_box.New(
			pay_caught_box_usecase.New(winsWallet, unlockStore, winRules), props.Metrics)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the rewards to boxes.v1.BoxClaimed: %w", err)
	}

	// What progression unlocked for each account, which the draws read while the gates are on.
	keptUnlocks, err := cpredis.Handle(ctx, streams,
		log_subscriber.New(keep_unlocks_handler.New(keep_unlocks_usecase.New(unlockStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to handle bonuses.v1.KeepUnlocks: %w", err)
	}
	levelUps, err := cpredis.Handle(ctx, streams,
		log_subscriber.New(pay_level_up_handler.New(pay_level_up_usecase.New(lastingWallet, winRules)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to handle bonuses.v1.PayLevelUp: %w", err)
	}
	forgotten, err := cpredis.Subscribe(ctx, streams, "bonuses-unlocks-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_unlocks_usecase.New(unlockStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the unlocks to player.v1.PlayerForgotten: %w", err)
	}

	// Not closers: the subscribers outlive the server.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		newPlayers, rightAnswers, boxesCaught, keptUnlocks, levelUps, forgotten)))

	service := bonusesv1controller.BonusService{
		GetGrantRulesHandler: get_grant_rules_handler.New(winRules),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return bonusesv1connect.NewBonusServiceHandler(service, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount bonuses.v1.BonusService: %w", err)
	}

	admin := bonusesv1controller.AdminService{
		GetRulesHandler: get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
		SaveRuleSetHandler: save_rule_set_handler.New(
			cprules.NewAuditedSave(cprules.NewSaveUseCase(db, ruleSets, ruleParts, playRules), props.Logger,
				cprules.SaveLog{Saved: "admin offer rule set saved", Refused: "admin offer rule set refused"})),
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
