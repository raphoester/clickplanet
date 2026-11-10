package acknowledge_warning_handler

import (
	"context"

	"connectrpc.com/connect"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/caller"
)

type Book interface {
	Acknowledge(ctx context.Context, account string) error
}

func New(book Book) AcknowledgeWarningHandler {
	return AcknowledgeWarningHandler{book: book}
}

type AcknowledgeWarningHandler struct {
	book Book
}

func (h AcknowledgeWarningHandler) AcknowledgeWarning(
	ctx context.Context,
	_ *connect.Request[antibotv1.AcknowledgeWarningRequest],
) (*connect.Response[antibotv1.AcknowledgeWarningResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}
	if err := h.book.Acknowledge(ctx, account); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&antibotv1.AcknowledgeWarningResponse{}), nil
}
