package backfill_reply_addresses_handler

import (
	"context"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type UseCase interface {
	Execute(ctx context.Context, dryRun bool) (notes.BackfillCounts, error)
}

func New(useCase UseCase) BackfillReplyAddressesHandler {
	return BackfillReplyAddressesHandler{useCase: useCase}
}

type BackfillReplyAddressesHandler struct {
	useCase UseCase
}

func (h BackfillReplyAddressesHandler) BackfillReplyAddresses(
	ctx context.Context,
	req *connect.Request[supportv1.BackfillReplyAddressesRequest],
) (*connect.Response[supportv1.BackfillReplyAddressesResponse], error) {
	counts, err := h.useCase.Execute(ctx, req.Msg.GetDryRun())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&supportv1.BackfillReplyAddressesResponse{
		Filled: uint32(counts.Filled), Guests: uint32(counts.Guests), NoAddress: uint32(counts.NoAddress), //nolint:gosec // counts of notes.
	}), nil
}
