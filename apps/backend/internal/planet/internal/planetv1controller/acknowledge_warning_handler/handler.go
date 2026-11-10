package acknowledge_warning_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Book interface {
	Acknowledge(ctx context.Context, payer clicks.Payer) error
}

func New(book Book) AcknowledgeWarningHandler {
	return AcknowledgeWarningHandler{book: book}
}

type AcknowledgeWarningHandler struct {
	book Book
}

func (h AcknowledgeWarningHandler) AcknowledgeWarning(
	ctx context.Context,
	_ *connect.Request[planetv1.AcknowledgeWarningRequest],
) (*connect.Response[planetv1.AcknowledgeWarningResponse], error) {
	if err := h.book.Acknowledge(ctx, clicks.PayerOf(ctx)); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.AcknowledgeWarningResponse{}), nil
}
