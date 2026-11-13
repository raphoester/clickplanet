// Package caller reads who a request comes from, in the quizzes' words: the address and the click token the session
// interceptors put on its context.
package caller

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

// Of is the caller as planet's ClickTried names it: the scope of its address, its account and whether it signed in.
func Of(ctx context.Context) quizzes.Caller {
	return quizzes.Caller{
		Scope:   cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account: quizzes.Account(cpctx.GetAccount(ctx)),
		Linked:  cpctx.GetLinked(ctx),
	}
}
