package rpc_scoring_standings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/rpc_scoring_standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubScoring struct {
	asked string
	times int
	table *scoringv1.GetTableResponse
	err   error
}

func (s *stubScoring) GetTable(
	_ context.Context, req *connect.Request[scoringv1.GetTableRequest],
) (*connect.Response[scoringv1.GetTableResponse], error) {
	s.asked = req.Msg.GetBookId()
	s.times++
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.table), nil
}

var (
	now = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	day = days.TestingDayAt(now)
)

func reader(scoring *stubScoring) *rpc_scoring_standings.Reader {
	return rpc_scoring_standings.New(scoring, cptime.NewFixedClock(now))
}

func TestTheStandingIsEachFlagsPointsInTheBookOfTheDaysSeason(t *testing.T) {
	scoring := &stubScoring{table: &scoringv1.GetTableResponse{
		Table:        []*scoringv1.TablePlace{{CountryId: "pl", Rank: 1, Points: 310}, {CountryId: "fr", Rank: 2, Points: 155}},
		OpenRoundIds: []string{day.String()},
	}}

	standing, err := reader(scoring).Standing(t.Context(), day)

	require.NoError(t, err)
	assert.Equal(t, "season-0", scoring.asked)
	assert.Equal(t, flagmissions.StandingSnapshot{Points: map[string]uint32{"pl": 310, "fr": 155}, Settled: true},
		standing.Snapshot(), "the day's own round is open")
}

func TestARoundOpenBeforeTheDayLeavesTheStandingUnsettled(t *testing.T) {
	for _, open := range [][]string{{"season-0/day-21"}, {"0192f5a0-7c00-7000-8000-000000000031", day.String()}} {
		standing, err := reader(&stubScoring{table: &scoringv1.GetTableResponse{OpenRoundIds: open}}).
			Standing(t.Context(), day)

		require.NoError(t, err)
		assert.False(t, standing.Settled(), "%v: the day before or the battle between is not scored yet", open)
	}
}

func TestABookNotOpenedYetHasNoPoints(t *testing.T) {
	standing, err := reader(&stubScoring{err: connect.NewError(connect.CodeNotFound, errors.New("no book"))}).
		Standing(t.Context(), day)

	require.NoError(t, err)
	assert.Equal(t, flagmissions.StandingSnapshot{Points: map[string]uint32{}, Settled: true}, standing.Snapshot())
}

func TestAScoringThatFailsIsAnError(t *testing.T) {
	_, err := reader(&stubScoring{err: connect.NewError(connect.CodeUnavailable, errors.New("down"))}).
		Standing(t.Context(), day)

	assert.Error(t, err)
}

func TestATableIsKeptASecondForTheReadsThatAskForADrawMeanwhile(t *testing.T) {
	scoring := &stubScoring{table: &scoringv1.GetTableResponse{OpenRoundIds: []string{"season-0/day-21"}}}
	clock := cptime.NewFixedClock(now)
	standings := rpc_scoring_standings.New(scoring, clock)

	for range 4 {
		_, err := standings.Standing(t.Context(), day)
		require.NoError(t, err)
		clock.Advance(400 * time.Millisecond)
	}
	assert.Equal(t, 2, scoring.times, "asked at 0 and at 1.2 s")
	scoring.table = &scoringv1.GetTableResponse{OpenRoundIds: []string{day.String()}}

	clock.Advance(time.Second)
	standing, err := standings.Standing(t.Context(), day)
	require.NoError(t, err)
	assert.True(t, standing.Settled(), "the day before was scored meanwhile")
}

func TestAnotherSeasonsBookIsAskedAtOnce(t *testing.T) {
	scoring := &stubScoring{table: &scoringv1.GetTableResponse{}}
	standings := reader(scoring)
	next, err := days.DayOf(days.DaySnapshot{
		ID: "season-1/day-1", Book: "season-1", Season: 1, Number: 1, StartsAt: now, EndsAt: now.Add(time.Hour),
	})
	require.NoError(t, err)

	_, err = standings.Standing(t.Context(), day)
	require.NoError(t, err)
	_, err = standings.Standing(t.Context(), next)
	require.NoError(t, err)

	assert.Equal(t, 2, scoring.times)
	assert.Equal(t, "season-1", scoring.asked)
}
