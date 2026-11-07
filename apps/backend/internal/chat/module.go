package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"

	moderationdata "github.com/raphoester/clickplanet.lol-backend/generated/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/postgres_announcement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/rpc_scoring_results"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_champion_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler/message_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_mute_handler/mute_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_raid_mode_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_raid_mode_handler/raid_mode_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler/messages_query"
	messages_authors "github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler/messages_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler/moderation_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mark_seen_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/remove_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/set_raid_mode_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unmute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/leave_room_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/claude_judge"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/postgres_conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/postgres_judgement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/redis_outbox_sender"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase/log_judgement"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/log_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_planet_flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase/log_prune"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/log_banned"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/muting_send_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/publishing_send_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/raiding_send_message"
	message_ids "github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/postgres_moderation_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/usecases/forget_moderation_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/postgres_mute_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase/audit_mute"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase/publishing_mute"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase/audit_unmute"
	mute_ids "github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/postgres_raid_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase/audit_set_raid_mode"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/postgres_reaction_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase/muting_react"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/postgres_removal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase/audit_remove"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/postgres_seen_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/forget_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/mark_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/account_muted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/allegiance_pledged_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/battle_ended_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/bomb_landed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/day_ended_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/flag_mission_done_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/fortified_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/player_reported_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/season_ended_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "chat"

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
	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the chat asks the player module who posts, scoring who won and planet who plays for a flag: %w",
			err)
	}
	player := playerv1connect.NewInternalServiceClient(internal, baseURL)
	rooms := messages.NewRooms(rpc_planet_flags.New(planetv1connect.NewInternalServiceClient(internal, baseURL)))
	results := rpc_scoring_results.New(scoringv1connect.NewInternalServiceClient(internal, baseURL))

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the chat to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the chat to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	storage := config.Storage.withDefaults()
	messageStore := postgres_message_store.New(db)
	reactionStore := postgres_reaction_store.New(db)
	announcementStore := postgres_announcement_store.New(db)
	seenStore := postgres_seen_store.New(db)
	muteStore := postgres_mute_store.New(db)
	removalStore := postgres_removal_store.New(db)
	moderationStore := postgres_moderation_store.New(db)
	judgementStore := postgres_judgement_store.New(db)
	muteBook := mutes.NewBook(muteStore, cptime.SystemClock{})
	raidStore := postgres_raid_store.New(db)
	raidBook := raids.NewBook(raidStore, cptime.SystemClock{})
	window := messages.NewWindow(storage.HistorySize, storage.Retention)
	updates := inprocess_feed.New(storage.SubscriberBuffer, props.Logger)

	prune := log_prune.New(
		prune_usecase.New(cptime.SystemClock{},
			prune_usecase.KeptFor(storage.Retention,
				messageStore, reactionStore, announcementStore, muteStore, removalStore, judgementStore),
			prune_usecase.KeptFor(storage.ModerationRetention, moderationStore)),
		props.Logger)

	announce := announce_usecase.New(announcementStore, updates)
	authors := log_authors.New(rpc_player_authors.New(player), props.Logger)
	limits := messages.NewLimits(config.Service.MaxTextLength).WithBanned(
		messages.NewBanned(slices.Concat(messages.DefaultBannedPhrases, config.Service.BannedPhrases)...))
	drafts := messages.NewDrafts(message_ids.Provider{}, cptime.SystemClock{}, limits, cpcountries.New(), rooms)
	thread := messages.NewThread(messageStore, cptime.SystemClock{}, window)

	bombs, err := cpredis.Subscribe(ctx, streams, "chat-announcements-bombs",
		log_subscriber.New(bomb_landed_subscriber.New(announce), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to planet.v1.BombLanded: %w", err)
	}

	fortifies, err := cpredis.Subscribe(ctx, streams, "chat-announcements-fortifies",
		log_subscriber.New(fortified_subscriber.New(announce), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to planet.v1.Fortified: %w", err)
	}

	flagMissions, err := cpredis.Subscribe(ctx, streams, "chat-announcements-flag-missions",
		log_subscriber.New(flag_mission_done_subscriber.New(announce), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to missions.v1.FlagMissionDone: %w", err)
	}

	deletions, err := cpredis.Subscribe(ctx, streams, "chat-seen-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_seen_usecase.New(seenStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to player.v1.PlayerForgotten: %w", err)
	}

	forgottenModeration, err := cpredis.Subscribe(ctx, streams, "chat-moderation-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(forget_moderation_usecase.New(moderationStore)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the moderation history to player.v1.PlayerForgotten: %w", err)
	}

	mutings, err := cpredis.Subscribe(ctx, streams, "chat-announcements-mutes",
		log_subscriber.New(account_muted_subscriber.New(announce_mute_usecase.New(authors, announce)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to chat.v1.AccountMuted: %w", err)
	}

	days, err := cpredis.Subscribe(ctx, streams, "chat-announcements-days",
		log_subscriber.New(day_ended_subscriber.New(announce_round_usecase.New(results, announce)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to gamemaster.v1.DayEnded: %w", err)
	}

	battles, err := cpredis.Subscribe(ctx, streams, "chat-announcements-battles",
		log_subscriber.New(battle_ended_subscriber.New(announce_round_usecase.New(results, announce)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to gamemaster.v1.BattleEnded: %w", err)
	}

	champions, err := cpredis.Subscribe(ctx, streams, "chat-announcements-champions",
		log_subscriber.New(season_ended_subscriber.New(announce_champion_usecase.New(results, announce)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to gamemaster.v1.SeasonEnded: %w", err)
	}

	judging := config.Judge.withDefaults()
	judge, err := judgeOf(judging)
	if err != nil {
		closeStores()
		return err
	}
	judgeCalls := cpratelimit.New("judge-calls", cpratelimit.Config{
		PerSecond: float64(judging.CallsPerHour) / time.Hour.Seconds(), Burst: judging.CallsPerHour,
	}, cptime.SystemClock{})
	props.Runners.Add(judgeCalls)
	judgeReport := log_judgement.New(judge_report_usecase.New(judge_report_usecase.Ports{
		Judgements:    judgementStore,
		Conversations: postgres_conversations.New(db),
		Authors:       authors,
		History:       moderationStore,
		Judge:         judge,
		Calls:         judgeCalls,
		Transactioner: db,
		Remover: remove_message_usecase.New(
			db, messageStore, removalStore, moderationStore, updates, window, cptime.SystemClock{}),
		Muter:  mute_usecase.New(db, muteStore, moderationStore, messageStore, mute_ids.Provider{}, cptime.SystemClock{}),
		Outbox: cpoutbox.New(db),
	}, window, cptime.SystemClock{}), props.Logger)
	reports, err := cpredis.Subscribe(ctx, streams, "chat-judge-reports",
		log_subscriber.New(player_reported_subscriber.New(judgeReport), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe the judge to support.v1.PlayerReported: %w", err)
	}

	departures, err := cpredis.Subscribe(ctx, streams, "chat-rooms",
		log_subscriber.New(allegiance_pledged_subscriber.New(leave_room_usecase.New(updates)), props.Logger))
	if err != nil {
		closeStores()
		return fmt.Errorf("failed to subscribe to planet.v1.AllegiancePledged: %w", err)
	}

	// Not closers: closers run before the runners stop, and the runners use both.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, prune_usecase.NewRunner(storage.PruneInterval, prune),
		cpredis.CloseAfter(streams, props.Logger, bombs, fortifies, flagMissions, deletions, forgottenModeration, mutings,
			days, battles, champions, departures, reports,
			cpoutbox.NewRelay("chat-outbox", db, redis_outbox_sender.New(streams, streams.Commands()), props.Logger))))

	messageLimiter := cpratelimit.New("message-limiter", config.RateLimiter, cptime.SystemClock{})
	props.Runners.Add(messageLimiter)

	reactionLimiter := cpratelimit.New("reaction-limiter", config.ReactionLimiter, cptime.SystemClock{})
	props.Runners.Add(reactionLimiter)

	seenLimiter := cpratelimit.New("seen-limiter", config.SeenLimiter, cptime.SystemClock{})
	props.Runners.Add(seenLimiter)

	blocklist, err := cpipblock.NewDenyList(config.BlockedIPs)
	if err != nil {
		return fmt.Errorf("failed to build the chat blocklist: %w", err)
	}

	chatService := chatv1controller.ChatService{
		SendMessageHandler: send_message_handler.New(muting_send_message.New(raiding_send_message.New(
			log_banned.New(publishing_send_message.New(send_message_usecase.New(
				messageStore, updates, authors, drafts, thread), streams), props.Logger),
			raidBook, authors), muteBook)),
		GetHistoryHandler: get_history_handler.New(history_query.NewPostgresQuery(
			db, cptime.SystemClock{}, storage.HistorySize, storage.Retention), rooms),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(updates, rooms, props.Server.StreamHeartbeat)),
		ReactHandler: react_handler.New(muting_react.New(react_usecase.New(
			messages.NewShowing(messageStore, rooms, cptime.SystemClock{}, window),
			reactionStore, updates, cptime.SystemClock{}), muteBook)),
		MarkSeenHandler: mark_seen_handler.New(mark_seen_usecase.New(seenStore, rooms, cptime.SystemClock{})),
	}

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewChatServiceHandler(chatService, options...)
	},
		chatv1controller.NewBlocklistInterceptor(blocklist),
		chatv1controller.NewRateLimitInterceptor(messageLimiter),
		chatv1controller.NewReactionRateLimitInterceptor(reactionLimiter),
		chatv1controller.NewSeenRateLimitInterceptor(seenLimiter),
		chatv1controller.NewSessionInterceptor(props.Sessions, cptime.SystemClock{}),
	)
	if err != nil {
		return err
	}

	adminService := chatv1controller.AdminService{
		MuteHandler: mute_handler.New(audit_mute.New(publishing_mute.New(
			mute_usecase.New(db, muteStore, moderationStore, messageStore, mute_ids.Provider{}, cptime.SystemClock{}),
			streams), props.Logger)),
		UnmuteHandler: unmute_handler.New(audit_unmute.New(
			unmute_usecase.New(db, muteStore, moderationStore, cptime.SystemClock{}), props.Logger)),
		GetMuteHandler: get_mute_handler.New(mute_query.NewPostgresQuery(db, cptime.SystemClock{})),
		RemoveMessageHandler: remove_message_handler.New(audit_remove.New(remove_message_usecase.New(
			db, messageStore, removalStore, moderationStore, updates, window, cptime.SystemClock{}), props.Logger)),
		GetMessageHandler: get_message_handler.New(message_query.NewPostgresQuery(db, cptime.SystemClock{}, storage.Retention)),
		ListMessagesHandler: list_messages_handler.New(messages_query.NewPostgresQuery(
			db, messages_authors.New(player), cptime.SystemClock{}, storage.Retention)),
		ListModerationHandler: list_moderation_handler.New(moderation_query.NewPostgresQuery(db)),
		SetRaidModeHandler: set_raid_mode_handler.New(audit_set_raid_mode.New(
			set_raid_mode_usecase.New(raidStore, cptime.SystemClock{}), props.Logger)),
		GetRaidModeHandler: get_raid_mode_handler.New(raid_mode_query.NewPostgresQuery(db, cptime.SystemClock{})),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount chat.v1.AdminService: %w", err)
	}

	props.Logger.Info("chat built", slog.String("schema", config.Database.Schema),
		slog.Bool("judge", judging.Anthropic.APIKey != ""), slog.String("judgeModel", judging.Model),
		slog.Int("judgeCallsPerHour", judging.CallsPerHour))

	return nil
}

type Config struct {
	Database cppg.Config
	Redis    cpredis.Config

	Storage StorageConfig
	// Keep the name: it sets the chat.service.* config keys.
	Service send_message_usecase.Config

	RateLimiter     cpratelimit.Config
	ReactionLimiter cpratelimit.Config
	SeenLimiter     cpratelimit.Config

	BlockedIPs []string

	// The judge of the reports support tells: off with no key.
	Judge JudgeConfig
}

type JudgeConfig struct {
	// ANTHROPIC_API_KEY. None leaves every report to the operator, and sends nothing.
	Anthropic cpanthropic.Config
	// Unset is claude_judge.DefaultModel, Claude Haiku 5.5.
	Model string
	// The most reports judged by the model in an hour, so a report flood cannot spend more; the rest is left open.
	CallsPerHour int
}

const defaultJudgeCallsPerHour = 10

func (c JudgeConfig) withDefaults() JudgeConfig {
	if c.Model == "" {
		c.Model = claude_judge.DefaultModel
	}
	if c.CallsPerHour <= 0 {
		c.CallsPerHour = defaultJudgeCallsPerHour
	}
	return c
}

// The judge on a client of Anthropic's API built here; with no key, one that sends nothing.
func judgeOf(config JudgeConfig) (judge_report_usecase.Judge, error) {
	client, err := cpanthropic.NewClient(config.Anthropic)
	if errors.Is(err, cpanthropic.ErrNoKey) {
		return claude_judge.Unkeyed{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build the judge's client of Anthropic's API: %w", err)
	}
	return claude_judge.New(client, config.Model, moderationdata.Policy()), nil
}

type StorageConfig struct {
	HistorySize int
	Retention   time.Duration
	// The moderation history's: what an operator did to an account outlives the messages it was about.
	ModerationRetention time.Duration
	PruneInterval       time.Duration
	SubscriberBuffer    int
}

const (
	defaultHistorySize         = 200
	defaultRetention           = 30 * 24 * time.Hour
	defaultModerationRetention = 90 * 24 * time.Hour
	defaultPruneInterval       = time.Hour
)

func (c StorageConfig) withDefaults() StorageConfig {
	if c.HistorySize <= 0 {
		c.HistorySize = defaultHistorySize
	}
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	if c.ModerationRetention <= 0 {
		c.ModerationRetention = defaultModerationRetention
	}
	if c.PruneInterval <= 0 {
		c.PruneInterval = defaultPruneInterval
	}
	return c
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("chat.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("chat.redis: %w", err)
	}
	return nil
}
