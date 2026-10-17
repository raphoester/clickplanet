package report_bug_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/report_bug_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/attachment"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/pagecontext"
)

type UseCase interface {
	Execute(ctx context.Context, in report_bug_usecase.In) error
}

func New(useCase UseCase) ReportBugHandler {
	return ReportBugHandler{useCase: useCase}
}

type ReportBugHandler struct {
	useCase UseCase
}

func (h ReportBugHandler) ReportBug(
	ctx context.Context,
	req *connect.Request[supportv1.ReportBugRequest],
) (*connect.Response[supportv1.ReportBugResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	files, err := attachment.IDs(req.Msg.GetAttachmentIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidAttachment)
	}

	err = h.useCase.Execute(ctx, report_bug_usecase.In{
		Account:     account,
		Text:        req.Msg.GetText(),
		Diagnostics: pagecontext.Diagnostics(req.Msg.GetContext(), req.Header()),
		Attachments: files,
	})
	switch {
	case errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidText)
	case errors.Is(err, notes.ErrInvalidAttachment):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidAttachment)
	case errors.Is(err, notes.ErrNotUploaded):
		return nil, connect.NewError(connect.CodeFailedPrecondition, notes.ErrNotUploaded)
	case err != nil:
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}

	return connect.NewResponse(&supportv1.ReportBugResponse{}), nil
}
