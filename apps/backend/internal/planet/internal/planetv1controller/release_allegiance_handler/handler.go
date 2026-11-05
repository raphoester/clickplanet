package release_allegiance_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

type Book interface {
	Release(ctx context.Context, account allegiances.Account) error
}

func New(book Book, lock func() allegiances.Lock) ReleaseAllegianceHandler {
	return ReleaseAllegianceHandler{book: book, lock: lock}
}

type ReleaseAllegianceHandler struct {
	book Book
	lock func() allegiances.Lock
}

func (h ReleaseAllegianceHandler) ReleaseAllegiance(
	ctx context.Context,
	req *connect.Request[planetv1.ReleaseAllegianceRequest],
) (*connect.Response[planetv1.ReleaseAllegianceResponse], error) {
	account, err := allegiances.AccountFrom(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.book.Release(ctx, account); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.ReleaseAllegianceResponse{Locked: bool(h.lock())}), nil
}
