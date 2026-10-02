// Package chat wires the live chat: its storage, its domain and its edge.
//
// Nothing here reaches into the tile game, and the tile game does not reach in
// here. The two share the process, the transport and the country list, and the
// country list arrives as an argument precisely so that sharing it costs no
// import between them.
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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/log_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase/log_prune"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/postgres_reaction_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/bomb_landed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "chat"

// bombLandedBuffer is how many bombs may wait on postgres before one goes unannounced. Bombs are rare: one per box.
const bombLandedBuffer = 256

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

	// Postgres is the chat's only copy: nothing is loaded at boot, and every read and write goes there.
	storage := config.Storage.withDefaults()
	messageStore := postgres_message_store.New(db)
	reactionStore := postgres_reaction_store.New(db)
	announcementStore := postgres_announcement_store.New(db)
	window := messages.Window{Size: storage.HistorySize, Retention: storage.Retention}
	updates := inprocess_feed.New(storage.SubscriberBuffer, props.Logger)

	prune := log_prune.New(
		prune_usecase.New(storage.Retention, cptime.SystemClock{}, messageStore, reactionStore, announcementStore),
		props.Logger)

	// Subscribed here, before any runner starts, so a bomb published at boot waits in the buffer. A full buffer
	// drops the announcement and counts it: the bomb still went off, the chat just does not say so.
	bombs, err := cpbootstrap.Subscribe(props.Events, "chat-announcements-bombs", bombLandedBuffer,
		log_subscriber.New(bomb_landed_subscriber.New(announce_usecase.New(announcementStore, updates)), props.Logger))
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to subscribe to planet.v1.BombLanded: %w", err)
	}

	// The pool closes after the runners stop, not as a closer: closers run first.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, prune_usecase.NewRunner(storage.PruneInterval, prune), bombs))

	messageLimiter := cpratelimit.New("message-limiter", config.RateLimiter, cptime.SystemClock{})
	props.Runners.Add(messageLimiter)

	reactionLimiter := cpratelimit.New("reaction-limiter", config.ReactionLimiter, cptime.SystemClock{})
	props.Runners.Add(reactionLimiter)

	blocklist, err := cpipblock.NewDenyList(config.BlockedIPs)
	if err != nil {
		return fmt.Errorf("failed to build the chat blocklist: %w", err)
	}

	// Who posts or reacts, a username or a guest code, comes from the player module over the internal listener:
	// one ask when a message is sent, so it can go out named; one per history for everyone in the window, the
	// people under its reactions included; and one per reaction, for the tally that goes back and out. A
	// failure to ask is logged, and the post is refused. Nothing here keeps a copy of a name.
	authors := log_authors.New(rpc_player_authors.New(props.Internal), props.Logger)

	chatService := chatv1controller.ChatService{
		SendMessageHandler: send_message_handler.New(send_message_usecase.New(
			messageStore, updates, cpcountries.New(), authors, cptime.SystemClock{}, config.Service)),
		GetHistoryHandler: get_history_handler.New(get_history_usecase.New(
			messageStore, reactionStore, announcementStore, authors, cptime.SystemClock{}, window)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(updates, props.Server.StreamHeartbeat)),
		ReactHandler: react_handler.New(react_usecase.New(
			messageStore, reactionStore, updates, authors, cptime.SystemClock{}, window)),
	}

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewChatServiceHandler(chatService, options...)
	},
		chatv1controller.NewBlocklistInterceptor(blocklist),
		chatv1controller.NewRateLimitInterceptor(messageLimiter),
		chatv1controller.NewReactionRateLimitInterceptor(reactionLimiter),
		// The key comes from auth over the internal listener, on the first token: this module holds no seed.
		chatv1controller.NewSessionInterceptor(rpc_session_verifier.New(props.Internal, props.Logger), cptime.SystemClock{}),
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
	// Named Service, not SendMessage, so the chat.service.* keys stay the same.
	Service send_message_usecase.Config

	RateLimiter cpratelimit.Config
	// ReactionLimiter throttles React per address. Its defaults, one a second and ten in hand, suit it.
	ReactionLimiter cpratelimit.Config

	BlockedIPs []string
}

// StorageConfig is how much of the chat is shown and kept. The keys predate the postgres-only storage.
type StorageConfig struct {
	// HistorySize is how many recent messages a joining client is shown, and can react to.
	HistorySize int
	// Retention is how long messages and reactions are kept: they are personal data.
	Retention     time.Duration
	PruneInterval time.Duration
	// SubscriberBuffer is how far one stream may fall behind before it misses updates.
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

// Validate refuses only a missing database: every other chat setting has a usable default.
func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("chat.database: %w", err)
	}
	return nil
}
