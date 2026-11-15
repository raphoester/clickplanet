// Package caller reads who a request comes from, in the boxes' words: the address and the click token the session
// interceptors put on its context.
package caller

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

// Of is the caller as planet's ClickTried names it: the scope of its address, its account and whether it signed in.
func Of(ctx context.Context) boxes.Caller {
	return boxes.Caller{
		Scope:   cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account: boxes.Account(cpctx.GetAccount(ctx)),
		Linked:  cpctx.GetLinked(ctx),
	}
}
