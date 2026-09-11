package chatv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
)

type MessageLimiter = connectutil.Limiter

type SenderBlocklist = connectutil.Blocklist

var (
	ErrTooManyMessages = errors.New("too many messages")
	ErrSenderBlocked   = errors.New("message refused")
)

func NewRateLimitInterceptor(limiter MessageLimiter) connect.Interceptor {
	return connectutil.NewRateLimitInterceptor(
		limiter,
		ErrTooManyMessages,
		nil, // the composer shows no allowance, so a refusal carries none
		chatv1connect.ChatServiceSendMessageProcedure,
	)
}

func NewBlocklistInterceptor(blocklist SenderBlocklist) connect.Interceptor {
	return connectutil.NewIPBlockInterceptor(
		blocklist,
		ErrSenderBlocked,
		nil,
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
	)
}
