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
	ErrTooManySeenMarks = errors.New("too many seen marks")
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

func NewSeenRateLimitInterceptor(limiter MessageLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManySeenMarks, chatv1connect.ChatServiceMarkSeenProcedure)
}

func NewBlocklistInterceptor(blocklist SenderBlocklist) connect.Interceptor {
	return cpconnect.NewIPBlockInterceptor(
		blocklist,
		ErrSenderBlocked,
		nil,
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
		chatv1connect.ChatServiceReactProcedure,
		chatv1connect.ChatServiceMarkSeenProcedure,
	)
}

type SenderSessionVerifier = cpconnect.SessionVerifier

func NewSessionInterceptor(verifier SenderSessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
		chatv1connect.ChatServiceReactProcedure,
		chatv1connect.ChatServiceMarkSeenProcedure,
	)
}

// Reading the chat, and marking it seen: a post or a reaction still needs the click token, which proves the Turnstile check.
func NewCookieReaderInterceptor(callers cpconnect.CookieCallers) connect.Interceptor {
	return cpconnect.NewCookieReaderInterceptor(callers,
		chatv1connect.ChatServiceGetHistoryProcedure,
		chatv1connect.ChatServiceMarkSeenProcedure,
	)
}
