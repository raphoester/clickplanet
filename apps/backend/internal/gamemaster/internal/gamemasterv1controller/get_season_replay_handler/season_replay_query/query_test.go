package season_replay_query_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_replay_handler/season_replay_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds  = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneStarts = seasonZeroEnds.Add(16 * time.Hour)
)

type countingPlanet struct {
	asked []*planetv1.GetMapHistoryRequest
}

func (p *countingPlanet) GetMapHistory(
	_ context.Context,
	req *connect.Request[planetv1.GetMapHistoryRequest],
) (*connect.Response[planetv1.GetMapHistoryResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(&planetv1.GetMapHistoryResponse{History: &planetv1.MapHistory{}}), nil
}

func calendar(t *testing.T) *inmemory_season_store.Store {
	t.Helper()
	store := inmemory_season_store.New()
	for _, planned := range []season.Snapshot{
		{Number: 0, Length: seasonZeroEnds.Sub(season.Origin), Finale: season.Finale{
			Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: schedule.Times(3),
		}},
		{Number: 1, Break: 16 * time.Hour, Length: 28 * 24 * time.Hour},
	} {
		require.NoError(t, store.Save(t.Context(), planned.Restore()))
	}
	return store
}

func TestAnEndedSeasonsReplayIsItsWindowEveryHalfHourAskedOnce(t *testing.T) {
	planet := &countingPlanet{}
	query := season_replay_query.New(calendar(t), planet, cptime.NewFixedClock(seasonOneStarts))

	for range 2 {
		_, err := query.Replay(t.Context(), season_replay_query.Part{Season: 0})
		require.NoError(t, err)
	}

	require.Len(t, planet.asked, 1, "an ended season's map no longer changes")
	assert.Equal(t, season.Origin, planet.asked[0].GetSince().AsTime())
	assert.Equal(t, seasonZeroEnds, planet.asked[0].GetUntil().AsTime())
	assert.Equal(t, 30*time.Minute, planet.asked[0].GetStep().AsDuration())
}

func TestTheFinalesReplayIsItsTwoHoursEveryTenSeconds(t *testing.T) {
	planet := &countingPlanet{}
	query := season_replay_query.New(calendar(t), planet, cptime.NewFixedClock(seasonOneStarts))

	_, err := query.Replay(t.Context(), season_replay_query.Part{Season: 0, Finale: true})
	require.NoError(t, err)

	require.Len(t, planet.asked, 1)
	assert.Equal(t, seasonZeroEnds.Add(-2*time.Hour), planet.asked[0].GetSince().AsTime())
	assert.Equal(t, 10*time.Second, planet.asked[0].GetStep().AsDuration())
}

func TestASeasonThatHasNotEndedOrHadNoFinaleHasNoReplay(t *testing.T) {
	planet := &countingPlanet{}
	query := season_replay_query.New(calendar(t), planet, cptime.NewFixedClock(seasonOneStarts.Add(29*24*time.Hour)))

	_, err := query.Replay(t.Context(), season_replay_query.Part{Season: 2})
	require.ErrorIs(t, err, season.ErrNotEnded)
	_, err = query.Replay(t.Context(), season_replay_query.Part{Season: 1, Finale: true})
	require.ErrorIs(t, err, season_replay_query.ErrNoFinale)
	assert.Empty(t, planet.asked)
}
