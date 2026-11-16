package call_rally_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetrefusal"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies/usecases/call_rally_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in call_rally_usecase.In) (rallies.Rally, error)
}

func New(useCase UseCase) CallRallyHandler {
	return CallRallyHandler{useCase: useCase}
}

type CallRallyHandler struct {
	useCase UseCase
}

func (h CallRallyHandler) CallRally(
	ctx context.Context,
	req *connect.Request[planetv1.CallRallyRequest],
) (*connect.Response[planetv1.CallRallyResponse], error) {
	point := req.Msg.GetPoint()
	rally, err := h.useCase.Execute(ctx, call_rally_usecase.In{
		Point: clicks.Vec3{X: point.GetX(), Y: point.GetY(), Z: point.GetZ()}, CountryID: req.Msg.GetCountryId(),
	})

	if refused, ok := planetrefusal.Refusal(err); ok {
		return nil, refused
	}
	if refused, ok := allegiancemessage.Refusal(err); ok {
		return nil, refused
	}

	switch {
	case err == nil:
		kept := rally.TakeSnapshot()
		return connect.NewResponse(&planetv1.CallRallyResponse{Rally: planetmessage.Rally(kept, lasts(kept))}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, rallies.ErrNoPoint):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, rallies.ErrRallyOn):
		return nil, preconditionWith(rallies.ErrRallyOn, &planetv1.RallyRefusal{})
	case errors.Is(err, call_rally_usecase.ErrNoRally):
		return nil, connect.NewError(connect.CodeNotFound, call_rally_usecase.ErrNoRally)
	default:
		return nil, err
	}
}

// A rally called is answered at once: all its time is left.
func lasts(rally rallies.Snapshot) time.Duration {
	return rally.EndsAt.Sub(rally.CalledAt)
}

func preconditionWith(err error, detail proto.Message) *connect.Error {
	refused := connect.NewError(connect.CodeFailedPrecondition, err)
	if detail, detailErr := connect.NewErrorDetail(detail); detailErr == nil {
		refused.AddDetail(detail)
	}
	return refused
}
