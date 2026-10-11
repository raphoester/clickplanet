package report_bug_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/report_bug_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/caller"
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

	page := req.Msg.GetContext()
	err = h.useCase.Execute(ctx, report_bug_usecase.In{
		Account: account,
		Text:    req.Msg.GetText(),
		Diagnostics: notes.Diagnostics{
			Build:          page.GetBuild(),
			ViewportWidth:  page.GetViewportWidth(),
			ViewportHeight: page.GetViewportHeight(),
			ConnectionDown: page.GetConnectionDown(),
			UserAgent:      req.Header().Get("User-Agent"),
		},
	})
	switch {
	case errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidText)
	case err != nil:
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}

	return connect.NewResponse(&supportv1.ReportBugResponse{}), nil
}
