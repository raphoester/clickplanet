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

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/adapters/primary/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/adapters/secondary/memory_chat_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/countries"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/secrets"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

const moduleName = "chat"

func NewModule(config Config) bootstrap.Module {
	return bootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(_ context.Context, props bootstrap.Props) error {
			return build(config, props)
		},
	}
}

func build(config Config, props bootstrap.Props) error {
	serviceConfig := config.Service
	if serviceConfig.TagSalt == "" {
		salt, err := secrets.RandomHex()
		if err != nil {
			return fmt.Errorf("failed to generate a chat tag salt: %w", err)
		}
		serviceConfig.TagSalt = salt
		props.Logger.Warning("no chat.service.tagSalt configured, generated a random one: sender tags will change on every restart")
	}

	storage := memory_chat_storage.New(config.Storage, xtime.ActualProvider{}, props.Logger)
	props.Runners.Add("chat-storage", storage.Run)

	service := chat_service.New(storage, countries.New(), xtime.ActualProvider{}, serviceConfig)

	messageLimiter := ratelimit.New(config.RateLimiter, xtime.ActualProvider{})
	props.Runners.Add("message-limiter", messageLimiter.Run)

	blocklist, err := ipblock.NewDenyList(config.BlockedIPs)
	if err != nil {
		return fmt.Errorf("failed to build the chat blocklist: %w", err)
	}

	err = props.RPC.Mount(chatv1connect.NewChatServiceHandler(
		chatv1controller.NewChatService(service, storage, props.Server.StreamHeartbeat),
		connect.WithInterceptors(
			chatv1controller.NewErrorInterceptor(props.Logger),
			chatv1controller.NewBlocklistInterceptor(blocklist),
			chatv1controller.NewRateLimitInterceptor(messageLimiter),
		),
	))
	if err != nil {
		return err
	}

	props.Logger.Info("chat enabled", lf.String("logPath", config.Storage.LogPath))

	return nil
}

type Config struct {
	// Off registers nothing, so /chat.v1.ChatService/ answers 404: the
	// unauthenticated public write endpoint does not exist rather than
	// existing and erroring.
	Enabled bool

	Storage memory_chat_storage.Config
	Service chat_service.Config

	RateLimiter ratelimit.Config

	BlockedIPs []string
}

// Validate has nothing to refuse: every chat setting has a usable default, so
// an unset one is a default rather than a misconfiguration. It exists so a
// check added later lands here and not in the binary's config.
func (Config) Validate() error {
	return nil
}
