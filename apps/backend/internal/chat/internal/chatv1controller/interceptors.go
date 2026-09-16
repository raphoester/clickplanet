package chatv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type MessageLimiter = cpconnect.Limiter

type SenderBlocklist = cpconnect.Blocklist

// MemberBans is the set an operator writes through chat.v1.AdminService.
type MemberBans interface {
	Banned(tag string) bool
}

var (
	ErrTooManyMessages = errors.New("too many messages")
	ErrSenderBlocked   = errors.New("message refused")
)

func NewRateLimitInterceptor(limiter MessageLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(
		limiter,
		ErrTooManyMessages,
		chatv1connect.ChatServiceSendMessageProcedure,
	)
}

func NewBlocklistInterceptor(blocklist SenderBlocklist) connect.Interceptor {
	return cpconnect.NewIPBlockInterceptor(
		blocklist,
		ErrSenderBlocked,
		nil,
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
	)
}

// NewBanInterceptor refuses a banned member's messages in the same words as every
// other refusal, and sits outside the limiter so one does not also spend a token.
func NewBanInterceptor(bans MemberBans, tagger messages.Tagger) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().Procedure != chatv1connect.ChatServiceSendMessageProcedure {
				return next(ctx, req)
			}

			ip := cpctx.GetSourceIP(ctx)
			if ip == "" {
				return next(ctx, req)
			}

			if bans.Banned(tagger.Of(ip)) {
				return nil, connect.NewError(connect.CodePermissionDenied, ErrSenderBlocked)
			}

			return next(ctx, req)
		}
	})
}
