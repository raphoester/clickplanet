package set_shown_in_clips_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_shown_in_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, in set_shown_in_clips_usecase.In) error
}

func New(useCase UseCase) SetShownInClipsHandler {
	return SetShownInClipsHandler{useCase: useCase}
}

type SetShownInClipsHandler struct {
	useCase UseCase
}

var ErrNoUsername = errors.New("only a player with a username may choose whether the clips show it")

func (h SetShownInClipsHandler) SetShownInClips(
	ctx context.Context,
	req *connect.Request[playerv1.SetShownInClipsRequest],
) (*connect.Response[playerv1.SetShownInClipsResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	err = h.useCase.Execute(ctx, set_shown_in_clips_usecase.In{Account: account, Shown: req.Msg.GetShown()})
	switch {
	case errors.Is(err, players.ErrNoProfile):
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrNoUsername)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.SetShownInClipsResponse{Shown: req.Msg.GetShown()}), nil
}
