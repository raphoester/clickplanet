// Package click_handler serves planet.v1.ClickService/Click: the proto message
// in, the click use case, the proto message out, and nothing else.
package click_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/clickbudget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
)

// UseCase is the port this handler calls, declared here the way every use case
// declares its own: what click_handler needs is a method, not a package's
// concrete type — which is also what lets a test stub the whole chain.
type UseCase interface {
	Execute(ctx context.Context, in click.In) (click.Out, error)
}

func New(useCase UseCase) ClickHandler {
	return ClickHandler{useCase: useCase}
}

type ClickHandler struct {
	useCase UseCase
}

// Click answers with what the throttle had left after letting this click
// through, so the meter on screen never lags what the server will enforce next.
//
// A caller error becomes a Connect code here rather than centrally: this is the
// only place that knows Click was asked, and so the only place that can say
// which of this procedure's refusals is the caller's fault.
func (h ClickHandler) Click(
	ctx context.Context,
	req *connect.Request[planetv1.ClickRequest],
) (*connect.Response[planetv1.ClickResponse], error) {
	out, err := h.useCase.Execute(ctx, click.In{
		TileID:    req.Msg.GetTileId(),
		CountryID: req.Msg.GetCountryId(),
	})
	if err != nil {
		return nil, toConnect(err, out)
	}

	res := &planetv1.ClickResponse{}
	if out.Limited {
		res.Budget = clickbudget.Encode(out.Budget)
	}

	return connect.NewResponse(res), nil
}
