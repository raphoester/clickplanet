package forgive_missed_day_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forgive_missed_day_usecase"
)

func New(useCase forgive_missed_day_usecase.Executor) ForgiveMissedDayHandler {
	return ForgiveMissedDayHandler{useCase: useCase}
}

type ForgiveMissedDayHandler struct {
	useCase forgive_missed_day_usecase.Executor
}

func (h ForgiveMissedDayHandler) ForgiveMissedDay(
	ctx context.Context,
	req *connect.Request[progressionv1.ForgiveMissedDayRequest],
) (*connect.Response[progressionv1.ForgiveMissedDayResponse], error) {
	account, err := accounts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	day, err := stats.DayFrom(req.Msg.GetDay())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	forgiven, err := h.useCase.Execute(ctx, forgive_missed_day_usecase.In{Account: account, Day: day, DryRun: req.Msg.GetDryRun()})
	switch {
	case errors.Is(err, stats.ErrNoStats):
		return nil, connect.NewError(connect.CodeNotFound, stats.ErrNoStats)
	case errors.Is(err, stats.ErrNotPast):
		return nil, connect.NewError(connect.CodeInvalidArgument, stats.ErrNotPast)
	case errors.Is(err, stats.ErrDayForgiven):
		return nil, connect.NewError(connect.CodeAlreadyExists, stats.ErrDayForgiven)
	case errors.Is(err, stats.ErrDayPlayed):
		return nil, connect.NewError(connect.CodeFailedPrecondition, stats.ErrDayPlayed)
	case errors.Is(err, stats.ErrJoinsNothing):
		return nil, connect.NewError(connect.CodeFailedPrecondition, stats.ErrJoinsNothing)
	case errors.Is(err, stats.ErrNoStreak):
		return nil, connect.NewError(connect.CodeFailedPrecondition, stats.ErrNoStreak)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	forgivenDays := forgiven.After.ForgivenDays()
	days := make([]string, 0, len(forgivenDays))
	for _, forgivenDay := range forgivenDays {
		days = append(days, forgivenDay.String())
	}
	return connect.NewResponse(&progressionv1.ForgiveMissedDayResponse{
		Before:       statsOf(forgiven.Before),
		After:        statsOf(forgiven.After),
		ForgivenDays: days,
	}), nil
}

func statsOf(counted stats.Stats) *progressionv1.Stats {
	return &progressionv1.Stats{
		TilesTaken:    counted.TilesTaken(),
		StreakCurrent: counted.Streak().Days(),
		StreakBest:    counted.StreakBest(),
		StreakLastDay: counted.Streak().LastDay().String(),
	}
}
