package link_members_handler

import (
	"context"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/link_members_usecase"
)

func New(useCase link_members_usecase.Executor) LinkMembersHandler {
	return LinkMembersHandler{useCase: useCase}
}

type LinkMembersHandler struct {
	useCase link_members_usecase.Executor
}

// The failures are counted here and logged by the use case's decorator, each with its reason.
func (h LinkMembersHandler) LinkMembers(
	ctx context.Context,
	_ *connect.Request[alertsv1.LinkMembersRequest],
) (*connect.Response[alertsv1.LinkMembersResponse], error) {
	out, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&alertsv1.LinkMembersResponse{
		Linked:  countOf(out.Linked),
		Outside: countOf(out.Outside),
		Failed:  countOf(len(out.Failed)),
		Named:   countOf(out.Named),
	}), nil
}

func countOf(n int) uint32 {
	return uint32(max(n, 0)) //nolint:gosec // a count of accounts, far below 2^32.
}
