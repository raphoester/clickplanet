package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/adapters/primary/chatv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/adapters/secondary/memory_chat_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/wspublisher"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// configureChatIfEnabled wires the live chat. Nothing is registered when it is
// off, so a disabled chat answers 404 rather than an error — and the public,
// unauthenticated write endpoint does not exist at all.
func (a *App) configureChatIfEnabled(_ context.Context) error {
	if !a.config.Chat.Enabled {
		return nil
	}

	serviceConfig := a.config.Chat.Service
	if serviceConfig.TagSalt == "" {
		salt, err := randomSalt()
		if err != nil {
			return fmt.Errorf("failed to generate a chat tag salt: %w", err)
		}
		serviceConfig.TagSalt = salt
		a.logger.Warning("no chat.service.tagSalt configured, generated a random one: sender tags will change on every restart")
	}

	chatStorage := memory_chat_storage.New(a.config.Chat.Storage, xtime.ActualProvider{}, a.logger)
	a.runners = append(a.runners, func() { chatStorage.Run(a.ctx) })

	service := chat_service.New(
		chatStorage,
		a.countryChecker(),
		xtime.ActualProvider{},
		serviceConfig,
	)

	messagesCh, err := chatStorage.Subscribe(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to chat messages: %w", err)
	}

	publisher := wspublisher.New(
		messagesCh,
		chatv1controller.MessageRoute,
		chatv1controller.EncodeMessage,
		a.logger,
	)
	a.runners = append(a.runners, publisher.Run)
	a.mountWS(publisher.DeclareRoutes)

	// Its own limiter, not the one the Click RPC uses: a message costs far more
	// than a click — it fans out to every client and lands in a log everyone
	// will read — so the two budgets have nothing to do with each other.
	messageLimiter := ratelimit.New(a.config.Chat.RateLimiter, xtime.ActualProvider{})
	a.runners = append(a.runners, func() { messageLimiter.Run(a.ctx) })

	a.mountRPC(chatv1connect.NewChatServiceHandler(
		chatv1controller.NewChatService(service),
		connect.WithInterceptors(
			chatv1controller.NewErrorInterceptor(a.logger),
			chatv1controller.NewGuardInterceptor(messageLimiter, a.config.Chat.BlockedIPs),
		),
	))

	a.logger.Info("chat enabled", lf.String("logPath", a.config.Chat.Storage.LogPath))

	return nil
}

func randomSalt() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
