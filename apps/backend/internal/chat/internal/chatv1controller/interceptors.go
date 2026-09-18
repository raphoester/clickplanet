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

// NewReactionRateLimitInterceptor throttles React on a bucket of its own: a reaction is cheaper than a message.
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

// NewSessionInterceptor reads a click token when the caller sends one, so a player posts and reacts as its
// account. It refuses nothing: a caller with no token, or a bad one, is a guest.
func NewSessionInterceptor(verifier SenderSessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
		chatv1connect.ChatServiceReactProcedure,
	)
}
