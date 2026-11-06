// Package chatoperator reads who calls an admin service off the context, as the admin gate put it there.
package chatoperator

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Nobody on the console listener, which has no gate.
func Of(ctx context.Context) moderation.Operator {
	operator := cpctx.GetOperator(ctx)
	return moderation.OperatorOf(operator.Account(), operator.Client())
}
