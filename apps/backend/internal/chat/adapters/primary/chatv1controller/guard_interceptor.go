package chatv1controller

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
)

// MessageLimiter answers whether a source IP may send another message now.
type MessageLimiter interface {
	Allow(key string) bool
}

// NewGuardInterceptor is the whole of chat's abuse control. SendMessage is a
// public, unauthenticated write that fans out to every connected client, so
// both checks run at the edge — ahead of decoding the message and well ahead of
// validating it, which makes a flood of malformed messages cost a sender
// exactly what a flood of well-formed ones does.
//
// blockedIPs cuts a sender off from the chat entirely, reads included: it is
// the config-driven way to stop an abusive address with a restart rather than a
// rebuild. The throttle applies to SendMessage alone — GetHistory is a single
// read on join, and limiting it would punish a page load.
func NewGuardInterceptor(limiter MessageLimiter, blockedIPs []string) connect.Interceptor {
	blocked := make(map[string]struct{}, len(blockedIPs))
	for _, ip := range blockedIPs {
		if trimmed := strings.TrimSpace(ip); trimmed != "" {
			blocked[trimmed] = struct{}{}
		}
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ip := ctxutil.GetSourceIP(ctx)

			if _, isBlocked := blocked[ip]; isBlocked {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("message refused"))
			}

			if req.Spec().Procedure != chatv1connect.ChatServiceSendMessageProcedure {
				return next(ctx, req)
			}

			if !limiter.Allow(ip) {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many messages"))
			}

			return next(ctx, req)
		}
	})
}
