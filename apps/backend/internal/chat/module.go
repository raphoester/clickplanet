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

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/adapters/primary/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/adapters/secondary/memory_chat_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsecrets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "chat"

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			return build(config, props)
		},
	}
}

func build(config Config, props cpbootstrap.Props) error {
	serviceConfig := config.Service
	if serviceConfig.TagSalt == "" {
		salt, err := cpsecrets.RandomHex()
		if err != nil {
			return fmt.Errorf("failed to generate a chat tag salt: %w", err)
		}
		serviceConfig.TagSalt = salt
		props.Logger.Warn("no chat.service.tagSalt configured, generated a random one: sender tags will change on every restart")
	}

	storage := memory_chat_storage.New(config.Storage, cptime.SystemClock{}, props.Logger)
	props.Runners.Add("chat-storage", storage.Run)

	service := chat_service.New(storage, cpcountries.New(), cptime.SystemClock{}, serviceConfig)

	messageLimiter := cpratelimit.New(config.RateLimiter, cptime.SystemClock{})
	props.Runners.Add("message-limiter", messageLimiter.Run)

	blocklist, err := cpipblock.NewDenyList(config.BlockedIPs)
	if err != nil {
		return fmt.Errorf("failed to build the chat blocklist: %w", err)
	}

	chatService := chatv1controller.NewChatService(service, storage, props.Server.StreamHeartbeat)

	err = props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return chatv1connect.NewChatServiceHandler(chatService, options...)
	},
		chatv1controller.NewBlocklistInterceptor(blocklist),
		chatv1controller.NewRateLimitInterceptor(messageLimiter),
	)
	if err != nil {
		return err
	}

	props.Logger.Info("chat enabled", slog.String("logPath", config.Storage.LogPath))

	return nil
}

type Config struct {
	// Off registers nothing, so /chat.v1.ChatService/ answers 404: the
	// unauthenticated public write endpoint does not exist rather than
	// existing and erroring.
	Enabled bool

	Storage memory_chat_storage.Config
	Service chat_service.Config

	RateLimiter cpratelimit.Config

	BlockedIPs []string
}

// Validate has nothing to refuse: every chat setting has a usable default, so
// an unset one is a default rather than a misconfiguration. It exists so a
// check added later lands here and not in the binary's config.
func (Config) Validate() error {
	return nil
}
