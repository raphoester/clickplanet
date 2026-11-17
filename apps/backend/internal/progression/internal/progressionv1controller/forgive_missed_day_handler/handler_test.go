package forgive_missed_day_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/forgive_missed_day_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forgive_missed_day_usecase"
)

var (
	ada  = accounts.AccountID{15: 1}
	noon = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
)

type stubUseCase struct {
	forgiven forgive_missed_day_usecase.Forgiven
	err      error
	asked    *[]forgive_missed_day_usecase.In
}

func (s stubUseCase) Execute(_ context.Context, in forgive_missed_day_usecase.In) (forgive_missed_day_usecase.Forgiven, error) {
	*s.asked = append(*s.asked, in)
	return s.forgiven, s.err
}

func newStub(forgiven forgive_missed_day_usecase.Forgiven, err error) stubUseCase {
	return stubUseCase{forgiven: forgiven, err: err, asked: &[]forgive_missed_day_usecase.In{}}
}

func forgive(t *testing.T, useCase stubUseCase, req *progressionv1.ForgiveMissedDayRequest) (*connect.Response[progressionv1.ForgiveMissedDayResponse], error) {
	t.Helper()

	return forgive_missed_day_handler.New(useCase).ForgiveMissedDay(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the tests read the error.
}

func TestTheAnswerTellsTheStreakBeforeAndAfterAndTheDaysForgiven(t *testing.T) {
	before := stats.StatsSnapshot{TilesTaken: 900, Streak: stats.StreakOf(2, stats.DayOf(noon)), StreakBest: 37}.Restore()
	after := stats.StatsSnapshot{
		TilesTaken: 900, Streak: stats.StreakOf(40, stats.DayOf(noon)), StreakBest: 40,
		ForgivenDays: []stats.Day{stats.DayOf(noon.AddDate(0, 0, -9)), stats.DayOf(noon.AddDate(0, 0, -2))},
	}.Restore()
	useCase := newStub(forgive_missed_day_usecase.Forgiven{Before: before, After: after}, nil)

	res, err := forgive(t, useCase, &progressionv1.ForgiveMissedDayRequest{AccountId: ada.String(), Day: "2026-11-15", DryRun: true})

	require.NoError(t, err)
	assert.True(t, proto.Equal(&progressionv1.ForgiveMissedDayResponse{
		Before:       &progressionv1.Stats{TilesTaken: 900, StreakCurrent: 2, StreakBest: 37, StreakLastDay: "2026-11-17"},
		After:        &progressionv1.Stats{TilesTaken: 900, StreakCurrent: 40, StreakBest: 40, StreakLastDay: "2026-11-17"},
		ForgivenDays: []string{"2026-11-08", "2026-11-15"},
	}, res.Msg), res.Msg)
	assert.Equal(t, []forgive_missed_day_usecase.In{{Account: ada, Day: stats.DayOf(noon.AddDate(0, 0, -2)), DryRun: true}}, *useCase.asked)
}

func TestAMalformedRequestIsInvalidAndAsksNothing(t *testing.T) {
	for name, req := range map[string]*progressionv1.ForgiveMissedDayRequest{
		"not an account": {AccountId: "ada", Day: "2026-11-15"},
		"not a day":      {AccountId: ada.String(), Day: "15/11/2026"},
		"no day":         {AccountId: ada.String()},
	} {
		t.Run(name, func(t *testing.T) {
			useCase := newStub(forgive_missed_day_usecase.Forgiven{}, nil)

			_, err := forgive(t, useCase, req)

			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.Empty(t, *useCase.asked)
		})
	}
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for refusal, code := range map[error]connect.Code{
		stats.ErrNoStats:      connect.CodeNotFound,
		stats.ErrNotPast:      connect.CodeInvalidArgument,
		stats.ErrDayForgiven:  connect.CodeAlreadyExists,
		stats.ErrDayPlayed:    connect.CodeFailedPrecondition,
		stats.ErrJoinsNothing: connect.CodeFailedPrecondition,
		stats.ErrNoStreak:     connect.CodeFailedPrecondition,
	} {
		t.Run(refusal.Error(), func(t *testing.T) {
			_, err := forgive(t, newStub(forgive_missed_day_usecase.Forgiven{}, fmt.Errorf("refused: %w", refusal)),
				&progressionv1.ForgiveMissedDayRequest{AccountId: ada.String(), Day: "2026-11-15"})

			assert.Equal(t, code, connect.CodeOf(err))
			var connectErr *connect.Error
			require.ErrorAs(t, err, &connectErr)
			assert.Equal(t, refusal.Error(), connectErr.Message(), "the sentence is the rule's")
		})
	}
}

func TestAnyOtherFailureIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("planet is down")

	_, err := forgive(t, newStub(forgive_missed_day_usecase.Forgiven{}, cause),
		&progressionv1.ForgiveMissedDayRequest{AccountId: ada.String(), Day: "2026-11-15"})

	assert.ErrorIs(t, err, cause)
}
