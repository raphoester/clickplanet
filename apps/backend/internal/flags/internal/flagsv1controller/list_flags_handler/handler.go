package list_flags_handler

import (
	"context"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
)

type Query interface {
	Flags(ctx context.Context) (*flagsv1.ListFlagsResponse, error)
}

func New(query Query) ListFlagsHandler {
	return ListFlagsHandler{query: query}
}

// ListFlagsHandler answers the players and the operator alike.
type ListFlagsHandler struct {
	query Query
}

func (h ListFlagsHandler) ListFlags(
	ctx context.Context,
	_ *connect.Request[flagsv1.ListFlagsRequest],
) (*connect.Response[flagsv1.ListFlagsResponse], error) {
	flags, err := h.query.Flags(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(flags), nil
}
