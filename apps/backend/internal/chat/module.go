package chat

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/postgres_announcement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler/history_query"
	history_authors "github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler/history_query/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mark_seen_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/log_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase/log_prune"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/publishing_send_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/postgres_reaction_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/postgres_seen_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/forget_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/mark_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/bomb_landed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "chat"

const bombLandedBuffer = 256

const accountDeletedBuffer = 256

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
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the chat to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	storage := config.Storage.withDefaults()
	messageStore := postgres_message_store.New(db)
	reactionStore := postgres_reaction_store.New(db)
	announcementStore := postgres_announcement_store.New(db)
	seenStore := postgres_seen_store.New(db)
	window := messages.NewWindow(storage.HistorySize, storage.Retention)
	updates := inprocess_feed.New(storage.SubscriberBuffer, props.Logger)

	prune := log_prune.New(
		prune_usecase.New(storage.Retention, cptime.SystemClock{}, messageStore, reactionStore, announcementStore),
		props.Logger)

	bombs, err := cpbootstrap.Subscribe(props.Events, "chat-announcements-bombs", bombLandedBuffer,
		log_subscriber.New(bomb_landed_subscriber.New(announce_usecase.New(announcementStore, updates)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.BombLanded: %w", err)
	}

	deletions, err := cpbootstrap.Subscribe(props.Events, "chat-seen", accountDeletedBuffer,
		log_subscriber.New(account_deleted_subscriber.New(forget_seen_usecase.New(seenStore)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to auth.v1.AccountDeleted: %w", err)
	}

	// Not a closer: closers run before the runners stop, and the runners use the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		prune_usecase.NewRunner(storage.PruneInterval, prune), bombs, deletions))

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

	authors := log_authors.New(rpc_player_authors.New(props.Internal), props.Logger)

	chatService := chatv1controller.ChatService{
		SendMessageHandler: send_message_handler.New(publishing_send_message.New(send_message_usecase.New(
			messageStore, updates, cpcountries.New(), authors, cptime.SystemClock{}, config.Service), props.Events)),
		GetHistoryHandler: get_history_handler.New(history_query.NewPostgresQuery(
			db, history_authors.New(props.Internal), cptime.SystemClock{}, storage.HistorySize, storage.Retention)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(updates, props.Server.StreamHeartbeat)),
		ReactHandler: react_handler.New(react_usecase.New(
			messageStore, reactionStore, updates, authors, cptime.SystemClock{}, window)),
		MarkSeenHandler: mark_seen_handler.New(mark_seen_usecase.New(seenStore, cptime.SystemClock{})),
	}

	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", "chat")))

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewChatServiceHandler(chatService, options...)
	},
		chatv1controller.NewBlocklistInterceptor(blocklist),
		chatv1controller.NewRateLimitInterceptor(messageLimiter),
		chatv1controller.NewReactionRateLimitInterceptor(reactionLimiter),
		chatv1controller.NewSeenRateLimitInterceptor(seenLimiter),
		chatv1controller.NewSessionInterceptor(verifier, cptime.SystemClock{}),
	)
	if err != nil {
		return err
	}

	props.Logger.Info("chat built", slog.String("schema", config.Database.Schema))

	return nil
}

type Config struct {
	Database cppg.Config

	Storage StorageConfig
	// Keep the name: it sets the chat.service.* config keys.
	Service send_message_usecase.Config

	RateLimiter     cpratelimit.Config
	ReactionLimiter cpratelimit.Config
	SeenLimiter     cpratelimit.Config

	BlockedIPs []string
}

type StorageConfig struct {
	HistorySize      int
	Retention        time.Duration
	PruneInterval    time.Duration
	SubscriberBuffer int
}

const (
	defaultHistorySize   = 200
	defaultRetention     = 30 * 24 * time.Hour
	defaultPruneInterval = time.Hour
)

func (c StorageConfig) withDefaults() StorageConfig {
	if c.HistorySize <= 0 {
		c.HistorySize = defaultHistorySize
	}
	if c.Retention <= 0 {
		c.Retention = defaultRetention
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
	return nil
}
