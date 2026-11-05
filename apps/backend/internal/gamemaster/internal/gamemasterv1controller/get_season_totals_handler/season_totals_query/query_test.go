package season_totals_query_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_totals_handler/season_totals_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds  = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneStarts = seasonZeroEnds.Add(16 * time.Hour)
	totals          = &planetv1.LedgerTotals{Takes: 812345, Bombs: 2100, Shields: 5300, Accounts: 1450}
)

type countingPlanet struct {
	asked []*planetv1.GetLedgerTotalsRequest
}

func (p *countingPlanet) GetLedgerTotals(
	_ context.Context,
	req *connect.Request[planetv1.GetLedgerTotalsRequest],
) (*connect.Response[planetv1.GetLedgerTotalsResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(&planetv1.GetLedgerTotalsResponse{Totals: totals}), nil
}

func calendar(t *testing.T) *inmemory_season_store.Store {
	t.Helper()
	store := inmemory_season_store.New()
	for _, planned := range []season.Snapshot{
		{Number: 0, Length: seasonZeroEnds.Sub(season.Origin)},
		{Number: 1, Break: 16 * time.Hour, Length: 28 * 24 * time.Hour},
	} {
		require.NoError(t, store.Save(t.Context(), planned.Restore()))
	}
	return store
}

func TestAnEndedSeasonsTotalsArePlanetsLedgerOverItsWindowAskedOnce(t *testing.T) {
	planet := &countingPlanet{}
	query := season_totals_query.New(calendar(t), planet, cptime.NewFixedClock(seasonOneStarts))

	first, err := query.Totals(t.Context(), 0)
	require.NoError(t, err)
	second, err := query.Totals(t.Context(), 0)
	require.NoError(t, err)

	assert.True(t, proto.Equal(totals, first), "%v", first)
	assert.True(t, proto.Equal(totals, second), "%v", second)
	require.Len(t, planet.asked, 1, "an ended season's totals no longer change")
	assert.Equal(t, season.Origin, planet.asked[0].GetSince().AsTime())
	assert.Equal(t, seasonZeroEnds, planet.asked[0].GetUntil().AsTime())
}

func TestASeasonThatHasNotEndedHasNoTotals(t *testing.T) {
	planet := &countingPlanet{}
	query := season_totals_query.New(calendar(t), planet, cptime.NewFixedClock(seasonOneStarts))

	_, err := query.Totals(t.Context(), 1)
	require.ErrorIs(t, err, season.ErrNotEnded)
	_, err = query.Totals(t.Context(), 7)
	require.ErrorIs(t, err, season.ErrNotEnded)
	assert.Empty(t, planet.asked)
}
