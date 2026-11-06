package chatv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
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

// The vendored VPN lists refuse what a mute refuses, a post and a reaction: a VPN still reads the chat. The answer is
// the blocklist's, so the sender is not told which list it is on; the counter is.
func NewVPNBlockInterceptor(blocklist SenderBlocklist, registerer prometheus.Registerer) connect.Interceptor {
	blocked := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "blocked_chat",
		Help: "Chat posts and reactions refused because the source address is in a blocked range",
	}, []string{"list"})

	return cpconnect.NewIPBlockInterceptor(
		blocklist,
		ErrSenderBlocked,
		func(list cpipblock.List) { blocked.WithLabelValues(string(list)).Inc() },
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceReactProcedure,
	)
}

type SenderSessionVerifier = cpconnect.SessionVerifier

func NewSessionInterceptor(verifier SenderSessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Attested(chatv1connect.ChatServiceSendMessageProcedure),
		cpconnect.Identified(chatv1connect.ChatServiceGetHistoryProcedure),
		cpconnect.Identified(chatv1connect.ChatServiceListenForEventsProcedure),
		cpconnect.Attested(chatv1connect.ChatServiceReactProcedure),
		cpconnect.Identified(chatv1connect.ChatServiceMarkSeenProcedure),
	)
}
