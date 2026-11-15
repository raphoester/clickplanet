package acknowledge_notice_handler

import (
	"context"

	"connectrpc.com/connect"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/caller"
)

type Book interface {
	Acknowledge(ctx context.Context, account string) error
}

func New(book Book) AcknowledgeNoticeHandler {
	return AcknowledgeNoticeHandler{book: book}
}

type AcknowledgeNoticeHandler struct {
	book Book
}

func (h AcknowledgeNoticeHandler) AcknowledgeNotice(
	ctx context.Context,
	_ *connect.Request[antibotv1.AcknowledgeNoticeRequest],
) (*connect.Response[antibotv1.AcknowledgeNoticeResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}
	if err := h.book.Acknowledge(ctx, account); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&antibotv1.AcknowledgeNoticeResponse{}), nil
}
