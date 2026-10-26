package antibot

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/inspect_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/unban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans/outbox_planet_enforcer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/commands/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/listener"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/observer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_choice_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_evidence_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/ban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/ban_player_usecase/audit_ban"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/inspect_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase/log_play_rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase/audit_save_rule_set"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/unban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/unban_player_usecase/audit_unban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "antibot"

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			clock := cptime.SystemClock{}

			db := cppg.New(config.Database)
			if err := db.ConnectCtx(ctx); err != nil {
				return fmt.Errorf("failed to connect the antibot to postgres: %w", err)
			}
			if err := db.Migrate(ctx, migrations.FS); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
			}

			streams := cpredis.New(config.Redis, props.Metrics)
			if err := streams.ConnectCtx(ctx); err != nil {
				_ = db.Close()
				return fmt.Errorf("failed to connect the antibot to redis: %w", err)
			}
			closeStores := func() {
				_ = streams.Close()
				_ = db.Close()
			}

			parts := guard.Parts()
			ruleSets := postgres_rule_set_store.New(db, parts)
			inForce := rules.NewInForce()

			book := bans.NewBook(rules.Reader(inForce, bans.Part), config.Evidence.WithDefaults().Retention, clock,
				postgres_ban_store.NewScopes(db), postgres_ban_store.NewAccounts(db),
				db, outbox_planet_enforcer.New(cpoutbox.New(db)))

			// The bans running at a deploy reach planet's refused callers with no hand migration.
			players := play_rules_usecase.New(ruleSets, postgres_choice_store.New(db), inForce, book)
			playRules := log_play_rules.New(players, props.Logger)
			if _, _, err := playRules.Execute(ctx); err != nil {
				closeStores()
				return fmt.Errorf("failed to play the antibot rules postgres keeps: %w", err)
			}
			chosen, err := cpredis.Handle(ctx, streams, play_rule_set_handler.New(playRules))
			if err != nil {
				closeStores()
				return fmt.Errorf("failed to handle antibot.v1.PlayRuleSet: %w", err)
			}

			marks := listener.NewMarks()
			watchdogs := guard.New(config.Config, inForce, clock, observer.New(props.Logger, props.Metrics), book,
				postgres_evidence_store.New(db), marks)
			if err := watchdogs.Load(ctx); err != nil {
				closeStores()
				return fmt.Errorf("failed to load the antibot state: %w", err)
			}
			observer.CountBans(watchdogs, props.Metrics)

			feeds, err := listen(ctx, streams, listener.New(watchdogs, marks))
			if err != nil {
				closeStores()
				return err
			}

			// Not closers: the evidence and the outbox keep being written until the last fact is handled.
			props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger,
				watchdogs.Hearing(feeds...), chosen, cpoutbox.NewRelay(moduleName, db, streams.Commands(), props.Logger))))

			admin := antibotv1controller.AdminService{
				BanPlayerHandler: ban_player_handler.New(
					audit_ban.New(ban_player_usecase.New(watchdogs), props.Logger)),
				UnbanPlayerHandler: unban_player_handler.New(
					audit_unban.New(unban_player_usecase.New(watchdogs), props.Logger)),
				InspectPlayerHandler: inspect_player_handler.New(inspect_player_usecase.New(watchdogs)),
				GetRulesHandler:      get_rules_handler.New(rule_sets_query.NewPostgresQuery(db), inForce),
				SaveRuleSetHandler: save_rule_set_handler.New(
					audit_save_rule_set.New(save_rule_set_usecase.New(ruleSets, parts, players), props.Logger)),
			}

			if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return antibotv1connect.NewAdminServiceHandler(admin, options...)
			}); err != nil {
				return fmt.Errorf("failed to mount antibot.v1.AdminService: %w", err)
			}

			internalService := antibotv1controller.InternalService{
				GetRuleSetNamesHandler: get_rule_set_names_handler.New(rule_set_names_query.NewPostgresQuery(db)),
			}
			if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				return antibotv1connect.NewInternalServiceHandler(internalService, options...)
			}); err != nil {
				return fmt.Errorf("failed to mount antibot.v1.InternalService: %w", err)
			}

			for _, alias := range antibotv1controller.PlanetAliases(admin) {
				if err := props.AdminRPC.Mount(alias); err != nil {
					return fmt.Errorf("failed to mount a planet.v1 path on antibot.v1: %w", err)
				}
			}

			return nil
		},
	}
}

func listen(ctx context.Context, streams *cpredis.Streams, heard *listener.Listener) ([]cpbootstrap.Runner, error) {
	var (
		feeds []cpbootstrap.Runner
		errs  []error
	)
	keep := func(feed cpbootstrap.Runner, err error) {
		feeds, errs = append(feeds, feed), append(errs, err)
	}

	keep(cpredis.Subscribe(ctx, streams, "antibot-clicks", heard.ClickTried()))
	keep(cpredis.Subscribe(ctx, streams, "antibot-map-reads", heard.MapRead()))
	keep(cpredis.Subscribe(ctx, streams, "antibot-streams", heard.StreamOpened()))
	keep(cpredis.Subscribe(ctx, streams, "antibot-boxes-caught", heard.BoxCaught()))
	keep(cpredis.Subscribe(ctx, streams, "antibot-boxes-lapsed", heard.BoxLapsed()))
	keep(cpredis.Subscribe(ctx, streams, "antibot-boxes-foreign", heard.ForeignBoxClaimed()))

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("failed to listen to planet: %w", err)
	}
	return feeds, nil
}
