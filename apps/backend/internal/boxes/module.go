package boxes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/box_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase/prom_claim_box"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/claim_box_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/rpc_planet_freeze"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/subscribers/click_tried_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName = "boxes"

	factsRelayInterval = 100 * time.Millisecond
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
		return fmt.Errorf("the boxes module calls planet: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the boxes to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the boxes to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	ruleParts := parts.All()
	ruleSets := cprules.NewPostgresStore(db, ruleParts)
	inForce := cprules.NewInForce[*boxesv1.Rules](clock)
	playRules := cprules.NewLoggedPlay(cprules.NewPlayUseCase(ruleSets, cprules.NewPostgresChoices(db), inForce, nil),
		props.Logger, cprules.PlayLog{Taken: "box rules in force", Failed: "failed to play the box rules"})
	if _, _, err := playRules.Execute(ctx); err != nil {
		closeStores()
		return fmt.Errorf("failed to play the boxes' rules postgres keeps: %w", err)
	}
	boxRules := cprules.Reader(inForce, boxes.Part)

	registry := boxes.New(config.Offers, clock, boxRules)
	told := facts.NewLog(clock)
	claim := take(registry, db, told, clock, props)

	// The bonuses module's group name, kept: the group goes on where the bonuses module left the stream.
	clicks, err := cpredis.Subscribe(ctx, streams, "bonuses-clicks",
		log_subscriber.New(click_tried_subscriber.New(registry), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the boxes to planet.v1.ClickTried: %w", err)
	}

	props.Runners.Add(registry)
	// Not closers: the subscribers, the outbox's last send and the relay's outlive the server.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
		clicks, cpoutbox.NewRelay("boxes-outbox", db, streams, props.Logger),
		facts.NewRelay(told, streams, factsRelayInterval))))

	freeze := rpc_planet_freeze.New(planet)
	service := boxesv1controller.BoxService{
		ClaimBoxHandler: claim_box_handler.New(claim),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(registry, props.Server.StreamHeartbeat)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return boxesv1connect.NewBoxServiceHandler(service, options...)
	},
		boxesv1controller.NewFreezeInterceptor(func(ctx context.Context) (bool, error) {
			if boxRules().Frozen {
				return true, nil
			}
			return freeze.Frozen(ctx)
		}),
		boxesv1controller.NewSessionInterceptor(props.Sessions, clock, props.Metrics),
		boxesv1controller.NewStreamSessionReader(props.Sessions, clock),
	); err != nil {
		return fmt.Errorf("failed to mount boxes.v1.BoxService: %w", err)
	}

	admin := boxesv1controller.AdminService{
		GetRulesHandler: get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
		SaveRuleSetHandler: save_rule_set_handler.New(
			cprules.NewAuditedSave(cprules.NewSaveUseCase(db, ruleSets, ruleParts, playRules), props.Logger,
				cprules.SaveLog{Saved: "admin box rule set saved", Refused: "admin box rule set refused"})),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return boxesv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount boxes.v1.AdminService: %w", err)
	}

	internalService := boxesv1controller.InternalService{
		PlayRuleSetHandler:     play_rule_set_handler.New(playRules),
		GetRuleSetNamesHandler: get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
		GetRuleSetHandler:      get_rule_set_handler.New(rule_set_query.NewPostgresQuery(db)),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return boxesv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount boxes.v1.InternalService: %w", err)
	}

	return nil
}

// take builds the claim, counts the boxes, and tells the antibot each box caught, lapsed or claimed by another scope:
// every metric keeps the name it had in planet and in the bonuses module.
func take(
	registry *boxes.Registry, db *cppg.Postgres, told *facts.Log, clock cptime.Clock, props cpbootstrap.Props,
) *prom_claim_box.Decorator {
	claim, counters := prom_claim_box.New(claim_box_usecase.New(registry, db, cpoutbox.New(db), clock), props.Metrics)

	facts := box_facts.New(told)
	registry.Observe(boxes.Report{
		Offered: counters.Offered.Inc,
		Lapsed: func(scope string) {
			counters.Lapsed.Inc()
			facts.Lapsed(scope)
		},
		Caught: func(scope string, after time.Duration) {
			counters.Caught.Observe(after.Seconds())
			facts.Caught(scope, after)
		},
		Foreign: func(scope string) {
			counters.Foreign.Inc()
			facts.Foreign(scope)
		},
	})

	return claim
}
