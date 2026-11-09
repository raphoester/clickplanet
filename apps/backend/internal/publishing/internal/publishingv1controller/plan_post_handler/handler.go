package plan_post_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/publishingwire"
)

type Query interface {
	Plan(ctx context.Context, platforms []calendar.Platform, notBefore, by time.Time) (*publishingv1.PlanPostResponse, error)
}

func New(query Query) PlanPostHandler {
	return PlanPostHandler{query: query}
}

type PlanPostHandler struct {
	query Query
}

func (h PlanPostHandler) PlanPost(
	ctx context.Context,
	req *connect.Request[publishingv1.PlanPostRequest],
) (*connect.Response[publishingv1.PlanPostResponse], error) {
	platforms, err := publishingwire.Platforms(req.Msg.GetPlatforms())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var notBefore, by time.Time
	if req.Msg.GetNotBefore() != nil {
		notBefore = req.Msg.GetNotBefore().AsTime()
	}
	if req.Msg.GetBy() != nil {
		by = req.Msg.GetBy().AsTime()
	}

	answer, err := h.query.Plan(ctx, platforms, notBefore, by)
	var busy *posts.BusyError
	switch {
	case errors.As(err, &busy):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(answer), nil
}
