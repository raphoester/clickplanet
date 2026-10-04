package chatv1controller

import (
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
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

type Callers interface {
	Caller(ctx context.Context, cookie string) (messages.AccountID, error)
}

var cookieProcedures = []string{chatv1connect.ChatServiceGetHistoryProcedure, chatv1connect.ChatServiceMarkSeenProcedure}

// After the session interceptor: a valid token names the caller and auth is not asked.
func NewCookieReaderInterceptor(callers Callers) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			cookie := req.Header().Get("Cookie")
			if !slices.Contains(cookieProcedures, req.Spec().Procedure) || cpctx.GetAccount(ctx) != "" || cookie == "" {
				return next(ctx, req)
			}

			account, err := callers.Caller(ctx, cookie)
			if err != nil || account == messages.NoAccount {
				return next(ctx, req)
			}
			return next(cpctx.AddAccountToContext(ctx, account.String()), req)
		}
	})
}
