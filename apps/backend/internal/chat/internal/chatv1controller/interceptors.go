package chatv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type MessageLimiter = cpconnect.Limiter

type SenderBlocklist = cpconnect.Blocklist

var (
	ErrTooManyMessages  = errors.New("too many messages")
	ErrTooManyReactions = errors.New("too many reactions")
	ErrSenderBlocked    = errors.New("message refused")
)

func NewRateLimitInterceptor(limiter MessageLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(
		limiter,
		ErrTooManyMessages,
		chatv1connect.ChatServiceSendMessageProcedure,
	)
}

func NewReactionRateLimitInterceptor(limiter MessageLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManyReactions, chatv1connect.ChatServiceReactProcedure)
}

func NewBlocklistInterceptor(blocklist SenderBlocklist) connect.Interceptor {
	return cpconnect.NewIPBlockInterceptor(
		blocklist,
		ErrSenderBlocked,
		nil,
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
		chatv1connect.ChatServiceReactProcedure,
	)
}

type SenderSessionVerifier = cpconnect.SessionVerifier

func NewSessionInterceptor(verifier SenderSessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Attested(chatv1connect.ChatServiceSendMessageProcedure),
		cpconnect.Identified(chatv1connect.ChatServiceGetHistoryProcedure),
		cpconnect.Attested(chatv1connect.ChatServiceReactProcedure),
	)
}
