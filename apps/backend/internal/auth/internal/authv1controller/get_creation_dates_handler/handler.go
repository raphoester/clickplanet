package get_creation_dates_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase interface {
	Execute(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID]time.Time, error)
}

func New(useCase UseCase) GetCreationDatesHandler {
	return GetCreationDatesHandler{useCase: useCase}
}

type GetCreationDatesHandler struct {
	useCase UseCase
}

func (h GetCreationDatesHandler) GetCreationDates(
	ctx context.Context,
	req *connect.Request[authv1.GetCreationDatesRequest],
) (*connect.Response[authv1.GetCreationDatesResponse], error) {
	asked := make([]accounts.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, value := range req.Msg.GetAccountIds() {
		account, err := accounts.AccountIDOf(value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	dates, err := h.useCase.Execute(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := &authv1.GetCreationDatesResponse{Dates: make([]*authv1.CreationDate, 0, len(dates))}
	for account, createdAt := range dates {
		res.Dates = append(res.Dates, &authv1.CreationDate{AccountId: account.String(), CreatedAtUnixMs: createdAt.UnixMilli()})
	}
	return connect.NewResponse(res), nil
}
