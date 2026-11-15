package get_play_days_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// Progression's backfill asks a page of its stats at a time.
const MaxAccounts = 500

var ErrTooManyAccounts = fmt.Errorf("more than %d accounts asked at once", MaxAccounts)

type Query interface {
	PlayDays(ctx context.Context, accounts []ledger.AccountID) (*planetv1.GetPlayDaysResponse, error)
}

func New(query Query) GetPlayDaysHandler {
	return GetPlayDaysHandler{query: query}
}

type GetPlayDaysHandler struct {
	query Query
}

func (h GetPlayDaysHandler) GetPlayDays(
	ctx context.Context,
	req *connect.Request[planetv1.GetPlayDaysRequest],
) (*connect.Response[planetv1.GetPlayDaysResponse], error) {
	asked := req.Msg.GetAccountIds()
	if len(asked) > MaxAccounts {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrTooManyAccounts)
	}
	accounts := make([]ledger.AccountID, 0, len(asked))
	for _, id := range asked {
		account, err := ledger.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		accounts = append(accounts, account)
	}
	if len(accounts) == 0 {
		return connect.NewResponse(&planetv1.GetPlayDaysResponse{}), nil
	}

	days, err := h.query.PlayDays(ctx, accounts)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(days), nil
}
