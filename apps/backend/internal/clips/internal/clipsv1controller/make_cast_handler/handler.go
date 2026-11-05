package make_cast_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type UseCase interface {
	Execute(ctx context.Context, since, until time.Time) (casts.Cast, []casts.NumberedTake, error)
}

func New(useCase UseCase) MakeCastHandler {
	return MakeCastHandler{useCase: useCase}
}

type MakeCastHandler struct {
	useCase UseCase
}

func (h MakeCastHandler) MakeCast(
	ctx context.Context,
	req *connect.Request[clipsv1.MakeCastRequest],
) (*connect.Response[clipsv1.MakeCastResponse], error) {
	var since, until time.Time
	if at := req.Msg.GetSince(); at != nil {
		since = at.AsTime()
	}
	if at := req.Msg.GetUntil(); at != nil {
		until = at.AsTime()
	}

	cast, takes, err := h.useCase.Execute(ctx, since, until)
	if errors.Is(err, casts.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&clipsv1.MakeCastResponse{Cast: castOf(cast, takes)}), nil
}

// The names and the numbers, never an account.
func castOf(cast casts.Cast, takes []casts.NumberedTake) *clipsv1.Cast {
	kept := cast.TakeSnapshot()
	wire := &clipsv1.Cast{
		Id: kept.ID.String(), Since: timestamppb.New(kept.Window.Since), Until: timestamppb.New(kept.Window.Until),
		Fighters: make([]*clipsv1.Fighter, 0, len(kept.Fighters)),
		Takes:    make([]*clipsv1.CastTake, 0, len(takes)),
	}
	for _, fighter := range kept.Fighters {
		wire.Fighters = append(wire.Fighters, &clipsv1.Fighter{Number: uint32(fighter.Number), Name: fighter.Name})
	}
	for _, take := range takes {
		wire.Takes = append(wire.Takes, &clipsv1.CastTake{
			At: timestamppb.New(take.At), TileId: take.Tile, CountryId: take.Country, PreviousCountryId: take.Previous,
			Fighter: uint32(take.Fighter),
		})
	}
	return wire
}
