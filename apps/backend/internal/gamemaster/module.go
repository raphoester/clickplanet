package gamemaster

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/arena_closed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/postgres_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/redis_arena_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/rpc_planet_arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_arena_usecase/log_hear_arena"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase/log_play_arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/add_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/delete_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/edit_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_handler/battle_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_schedule_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_schedule_handler/schedule_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_replay_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_replay_handler/season_replay_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_totals_handler/season_totals_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_seasons_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_seasons_handler/seasons_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_timeline_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler/turn_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler/inprocess_turn_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler/inprocess_turn_feed/log_turn_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/book_closed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/book_missed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/postgres_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/redis_scoring_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/round_missed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/round_scored_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_book_usecase/log_hear_book"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_book_usecase/log_hear_missed_book"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_round_usecase/log_hear_missed_round"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_round_usecase/log_hear_round"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase/log_play_plan"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase/log_play_scoring"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase/log_tell_starts"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/redis_rule_set_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_antibot_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_planet_rule_sets"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/add_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/add_entry_usecase/audit_add_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/delete_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/delete_entry_usecase/audit_delete_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/edit_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/edit_entry_usecase/audit_edit_entry"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/play_schedule_usecase/log_play_schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/postgres_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/redis_map_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/add_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/add_season_usecase/audit_add_season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/delete_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/delete_season_usecase/audit_delete_season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/edit_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/edit_season_usecase/audit_edit_season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase/log_play_resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	moduleName    = "gamemaster"
	scheduleCheck = time.Second
	// sendAgain is how long a close waits for its answer, and the turn for its next send, before it is sent again.
	sendAgain = time.Minute
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
	cutoff := season.Cutoff(config.DailyCutoff)

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the gamemaster module asks planet and the antibot which rule sets they keep: %w", err)
	}
	keepers := []schedule.Keeper{
		rpc_planet_rule_sets.New(planetv1connect.NewInternalServiceClient(internal, baseURL)),
		rpc_antibot_rule_sets.New(antibotv1connect.NewInternalServiceClient(internal, baseURL)),
	}

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the gamemaster module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the gamemaster module to redis: %w", err)
	}

	entries := postgres_entry_store.New(db)
	seasons := postgres_season_store.New(db)
	ruleSchedule := play_schedule_usecase.NewRunner(scheduleCheck, log_play_schedule.New(
		play_schedule_usecase.New(entries, seasons, redis_rule_set_commands.New(streams.Commands()), clock, sendAgain),
		props.Logger))
	heard := postgres_moment_store.New(db)
	judged := postgres_arena_store.New(db)
	planner := moments.NewPlanner(seasons, entries, heard, judged, cutoff)
	told := outbox_moment_events.New(cpoutbox.New(db))
	planPlay := play_plan_usecase.NewRunner(scheduleCheck, log_play_plan.New(play_plan_usecase.New(planner,
		log_play_resets.New(play_resets_usecase.New(redis_map_commands.New(streams.Commands()), clock), props.Logger),
		log_play_arenas.New(play_arenas_usecase.New(judged,
			rpc_planet_arenas.New(planetv1connect.NewInternalServiceClient(internal, baseURL)),
			redis_arena_commands.New(streams.Commands()), clock, sendAgain), props.Logger),
		log_play_scoring.New(play_scoring_usecase.New(heard, redis_scoring_commands.New(streams.Commands()), clock, sendAgain),
			props.Logger),
		log_tell_starts.New(tell_starts_usecase.New(heard, told, db, clock), props.Logger),
	), props.Logger))
	rounds, err := cpredis.Subscribe(ctx, streams, "gamemaster-rounds", round_scored_subscriber.New(
		log_hear_round.New(hear_round_usecase.New(planner, heard, told, db), props.Logger)))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to scoring.v1.RoundScored: %w", err)
	}
	books, err := cpredis.Subscribe(ctx, streams, "gamemaster-books", book_closed_subscriber.New(
		log_hear_book.New(hear_book_usecase.New(planner, heard, told, db), props.Logger)))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to scoring.v1.BookClosed: %w", err)
	}
	missedRounds, err := cpredis.Subscribe(ctx, streams, "gamemaster-rounds-missed", round_missed_subscriber.New(
		log_hear_missed_round.New(hear_missed_round_usecase.New(heard), props.Logger)))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to scoring.v1.RoundMissed: %w", err)
	}
	missedBooks, err := cpredis.Subscribe(ctx, streams, "gamemaster-books-missed", book_missed_subscriber.New(
		log_hear_missed_book.New(hear_missed_book_usecase.New(heard), props.Logger)))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to scoring.v1.BookMissed: %w", err)
	}
	arenas, err := cpredis.Subscribe(ctx, streams, "gamemaster-arenas-closed", arena_closed_subscriber.New(
		log_hear_arena.New(hear_arena_usecase.New(judged), props.Logger)))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.ArenaClosed: %w", err)
	}

	turns := turn_query.NewPostgresQuery(db, clock, cutoff)
	turnFeed := inprocess_turn_feed.New(log_turn_reader.New(turns, props.Logger))
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, turnFeed, cpredis.CloseAfter(streams, props.Logger,
		ruleSchedule, planPlay, rounds, books, missedRounds, missedBooks, arenas, cpoutbox.NewRelay("gamemaster-outbox", db, streams, props.Logger))))

	admin := gamemasterv1controller.AdminService{
		GetScheduleHandler: get_schedule_handler.New(schedule_query.NewPostgresQuery(db, clock)),
		AddEntryHandler: add_entry_handler.New(audit_add_entry.New(
			add_entry_usecase.New(entries, planner, keepers, uuid_id_provider.Provider{}, clock), props.Logger)),
		EditEntryHandler: edit_entry_handler.New(audit_edit_entry.New(
			edit_entry_usecase.New(entries, planner, keepers, clock), props.Logger)),
		DeleteEntryHandler: delete_entry_handler.New(audit_delete_entry.New(
			delete_entry_usecase.New(entries, clock), props.Logger)),
		GetSeasonsHandler: get_seasons_handler.New(seasons_query.NewPostgresQuery(db, cutoff)),
		AddSeasonHandler: add_season_handler.New(audit_add_season.New(
			add_season_usecase.New(seasons, entries, keepers, cutoff, clock), props.Logger)),
		EditSeasonHandler: edit_season_handler.New(audit_edit_season.New(
			edit_season_usecase.New(seasons, entries, keepers, cutoff, clock), props.Logger)),
		DeleteSeasonHandler: delete_season_handler.New(audit_delete_season.New(
			delete_season_usecase.New(seasons, clock), props.Logger)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return gamemasterv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount gamemaster.v1.AdminService: %w", err)
	}

	internalService := gamemasterv1controller.InternalService{
		GetBattleHandler:   get_battle_handler.New(battle_query.NewPostgresQuery(db)),
		GetTimelineHandler: get_timeline_handler.New(turns),
	}
	if err := props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return gamemasterv1connect.NewInternalServiceHandler(internalService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount gamemaster.v1.InternalService: %w", err)
	}

	public := gamemasterv1controller.GamemasterService{
		GetTurnHandler:         get_turn_handler.New(turns),
		ListenForEventsHandler: listen_for_events_handler.New(turnFeed, props.Server.StreamHeartbeat),
		GetSeasonTotalsHandler: get_season_totals_handler.New(
			season_totals_query.New(seasons, planetv1connect.NewInternalServiceClient(internal, baseURL), clock)),
		GetSeasonReplayHandler: get_season_replay_handler.New(
			season_replay_query.New(seasons, planetv1connect.NewInternalServiceClient(internal, baseURL), clock)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return gamemasterv1connect.NewGamemasterServiceHandler(public, options...)
	}, gamemasterv1controller.NewCacheInterceptor()); err != nil {
		return fmt.Errorf("failed to mount gamemaster.v1.GamemasterService: %w", err)
	}

	props.Logger.Info("gamemaster built", slog.Duration("daily_cutoff", config.DailyCutoff),
		slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	// DailyCutoff is the time of day, UTC, every day of a season ends at.
	DailyCutoff time.Duration
	Database    cppg.Config
	Redis       cpredis.Config
}

func (c Config) Validate() error {
	if err := season.Cutoff(c.DailyCutoff).Check(); err != nil {
		return fmt.Errorf("gamemaster.dailyCutoff: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("gamemaster.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("gamemaster.redis: %w", err)
	}
	return nil
}
