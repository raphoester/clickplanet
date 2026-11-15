package backfill_days_played_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/backfill_days_played_usecase"
)

func New(useCase backfill_days_played_usecase.Executor) BackfillDaysPlayedHandler {
	return BackfillDaysPlayedHandler{useCase: useCase}
}

type BackfillDaysPlayedHandler struct {
	useCase backfill_days_played_usecase.Executor
}

func (h BackfillDaysPlayedHandler) BackfillDaysPlayed(
	ctx context.Context,
	req *connect.Request[progressionv1.BackfillDaysPlayedRequest],
) (*connect.Response[progressionv1.BackfillDaysPlayedResponse], error) {
	backfilled, err := h.useCase.Execute(ctx, req.Msg.GetDryRun())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&progressionv1.BackfillDaysPlayedResponse{
		Accounts: uint32(backfilled.Accounts), //nolint:gosec // counted from the stats' rows, never negative or past 2^32.
		Changed:  uint32(backfilled.Changed),  //nolint:gosec // as above.
	}), nil
}
