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

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/rpc_session_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/rpc_player_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/log_authors"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
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
	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the chat to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	storage := inmemory_message_storage.New(
		config.Storage, postgres_message_store.New(db), cptime.SystemClock{}, props.Logger)
	if err := storage.Load(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to load the chat: %w", err)
	}
	// The pool closes after the runner stops, not as a closer: closers run first.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, storage))

	messageLimiter := cpratelimit.New("message-limiter", config.RateLimiter, cptime.SystemClock{})
	props.Runners.Add(messageLimiter)

	blocklist, err := cpipblock.NewDenyList(config.BlockedIPs)
	if err != nil {
		return fmt.Errorf("failed to build the chat blocklist: %w", err)
	}

	// Who posts, username and tag, comes from the player module over the internal listener. A failure to ask is
	// logged, and the message is refused.
	authors := log_authors.New(rpc_player_authors.New(props.Internal), props.Logger)

	chatService := chatv1controller.ChatService{
		SendMessageHandler: send_message_handler.New(
			send_message_usecase.New(storage, cpcountries.New(), authors, cptime.SystemClock{}, config.Service)),
		GetHistoryHandler: get_history_handler.New(get_history_usecase.New(storage)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(storage, props.Server.StreamHeartbeat)),
	}

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewChatServiceHandler(chatService, options...)
	},
		chatv1controller.NewBlocklistInterceptor(blocklist),
		chatv1controller.NewRateLimitInterceptor(messageLimiter),
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

	Storage inmemory_message_storage.Config
	// Named Service, not SendMessage, so the chat.service.* keys stay the same.
	Service send_message_usecase.Config

	RateLimiter cpratelimit.Config

	BlockedIPs []string
}

// Validate refuses only a missing database: every other chat setting has a usable default.
func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("chat.database: %w", err)
	}
	return nil
}
