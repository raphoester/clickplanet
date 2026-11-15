package search_players_by_email_handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

const (
	defaultLimit = 50
	maxLimit     = 500
)

var ErrNoText = errors.New("type a part of an address")

type Query interface {
	Matches(ctx context.Context, text string, limit uint32, offset uint32) (*playerv1.SearchPlayersByEmailResponse, error)
}

func New(query Query) SearchPlayersByEmailHandler {
	return SearchPlayersByEmailHandler{query: query}
}

type SearchPlayersByEmailHandler struct {
	query Query
}

func (h SearchPlayersByEmailHandler) SearchPlayersByEmail(
	ctx context.Context,
	req *connect.Request[playerv1.SearchPlayersByEmailRequest],
) (*connect.Response[playerv1.SearchPlayersByEmailResponse], error) {
	text := strings.TrimSpace(req.Msg.GetText())
	if text == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoText)
	}

	limit := req.Msg.GetLimit()
	if limit == 0 {
		limit = defaultLimit
	}

	matches, err := h.query.Matches(ctx, text, min(limit, maxLimit), req.Msg.GetOffset())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(matches)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
