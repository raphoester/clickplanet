package search_emails_handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
)

// The most holders one page holds, and what a limit of 0 asks.
const MaxLimit = 500

var ErrNoText = errors.New("type a part of an address")

type Query interface {
	Holders(ctx context.Context, text string, limit int, offset int) (*authv1.SearchEmailsResponse, error)
}

func New(query Query) SearchEmailsHandler {
	return SearchEmailsHandler{query: query}
}

type SearchEmailsHandler struct {
	query Query
}

func (h SearchEmailsHandler) SearchEmails(
	ctx context.Context,
	req *connect.Request[authv1.SearchEmailsRequest],
) (*connect.Response[authv1.SearchEmailsResponse], error) {
	text := strings.TrimSpace(req.Msg.GetText())
	if text == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoText)
	}
	limit := int(req.Msg.GetLimit())
	if limit == 0 || limit > MaxLimit {
		limit = MaxLimit
	}

	found, err := h.query.Holders(ctx, text, limit, int(req.Msg.GetOffset()))
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(found), nil
}
