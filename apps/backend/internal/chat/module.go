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

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/inmemory_ban_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/postgres_ban_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/ban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/ban_member_usecase/audit_ban"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/list_bans_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/unban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/unban_member_usecase/audit_unban"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/ban_member_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_bans_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unban_member_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsecrets"
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
	serviceConfig := config.Service
	if serviceConfig.TagSalt == "" {
		salt, err := cpsecrets.RandomHex()
		if err != nil {
			return fmt.Errorf("failed to generate a chat tag salt: %w", err)
		}
		serviceConfig.TagSalt = salt
		props.Logger.Warn("no chat.service.tagSalt configured, generated a random one: sender tags will change on every restart")
	}

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

	banStorage := inmemory_ban_storage.New(postgres_ban_store.New(db), props.Logger)
	if err := banStorage.Load(ctx); err != nil {
		return fmt.Errorf("failed to load the chat bans: %w", err)
	}

	tagger := messages.NewTagger(serviceConfig.TagSalt)

	messageLimiter := cpratelimit.New("message-limiter", config.RateLimiter, cptime.SystemClock{})
	props.Runners.Add(messageLimiter)

	blocklist, err := cpipblock.NewDenyList(config.BlockedIPs)
	if err != nil {
		return fmt.Errorf("failed to build the chat blocklist: %w", err)
	}

	chatService := chatv1controller.ChatService{
		SendMessageHandler: send_message_handler.New(
			send_message_usecase.New(storage, cpcountries.New(), tagger, cptime.SystemClock{}, serviceConfig)),
		GetHistoryHandler: get_history_handler.New(get_history_usecase.New(storage, banStorage)),
		ListenForEventsHandler: listen_for_events_handler.New(
			listen_for_events_usecase.New(storage, props.Server.StreamHeartbeat)),
	}

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewChatServiceHandler(chatService, options...)
	},
		chatv1controller.NewBlocklistInterceptor(blocklist),
		chatv1controller.NewBanInterceptor(banStorage, tagger),
		chatv1controller.NewRateLimitInterceptor(messageLimiter),
	)
	if err != nil {
		return err
	}

	adminService := chatv1controller.AdminService{
		BanMemberHandler: ban_member_handler.New(audit_ban.New(
			ban_member_usecase.New(banStorage, storage, cptime.SystemClock{}), props.Logger)),
		UnbanMemberHandler: unban_member_handler.New(audit_unban.New(
			unban_member_usecase.New(banStorage), props.Logger)),
		ListBansHandler: list_bans_handler.New(list_bans_usecase.New(banStorage)),
	}

	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
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
