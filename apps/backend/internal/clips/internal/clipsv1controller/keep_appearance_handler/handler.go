package keep_appearance_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type UseCase interface {
	Execute(ctx context.Context, cast casts.CastID, fighters []casts.Number, post appearances.Post) (int, error)
}

func New(useCase UseCase) KeepAppearanceHandler {
	return KeepAppearanceHandler{useCase: useCase}
}

type KeepAppearanceHandler struct {
	useCase UseCase
}

func (h KeepAppearanceHandler) KeepAppearance(
	ctx context.Context,
	req *connect.Request[clipsv1.KeepAppearanceRequest],
) (*connect.Response[clipsv1.KeepAppearanceResponse], error) {
	cast, err := casts.CastIDOf(req.Msg.GetCastId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	post, err := postOf(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	fighters := make([]casts.Number, 0, len(req.Msg.GetFighters()))
	for _, number := range req.Msg.GetFighters() {
		fighters = append(fighters, casts.Number(number))
	}

	kept, err := h.useCase.Execute(ctx, cast, fighters, post)
	switch {
	case errors.Is(err, casts.ErrNoCast):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, casts.ErrUnknownFighter):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&clipsv1.KeepAppearanceResponse{Kept: uint32(kept)}), nil //nolint:gosec // the players of one post.
}

func postOf(msg *clipsv1.KeepAppearanceRequest) (appearances.Post, error) {
	id, err := appearances.PostIDOf(msg.GetPostId())
	if err != nil {
		return appearances.Post{}, err //nolint:wrapcheck // the domain named what is wrong.
	}
	platforms := make([]appearances.Platform, 0, len(msg.GetPlatforms()))
	for _, name := range msg.GetPlatforms() {
		platform, err := appearances.PlatformOf(name)
		if err != nil {
			return appearances.Post{}, err //nolint:wrapcheck // the domain named what is wrong.
		}
		platforms = append(platforms, platform)
	}
	var scheduledAt time.Time
	if at := msg.GetScheduledAt(); at != nil {
		scheduledAt = at.AsTime()
	}
	return appearances.PostOf(id, msg.GetTitle(), platforms, scheduledAt) //nolint:wrapcheck // the domain named what is wrong.
}
