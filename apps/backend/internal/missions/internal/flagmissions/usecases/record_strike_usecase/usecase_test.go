package record_strike_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/inmemory_flag_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/usecases/record_strike_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}, nil
}

type noPledges struct{}

func (noPledges) FlagOf(context.Context, missions.AccountID) (string, error) { return "", nil }

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	at       = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
)

func TestTheShieldsBrokenFinishTheMissionAndGiveTheFlagItsPointsOnce(t *testing.T) {
	store, outbox := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}
	useCase := record_strike_usecase.New(calendar, flagmissions.NewBriefing(store, fixedPlanet{}, noPledges{}), store,
		cpatomic.FakeTransactioner{}, outbox)
	drawn := flagmissions.Draw(calendar.DayOf(at), "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}},
		flagmissions.Day{})
	status, ok := drawn.Status()
	require.True(t, ok)
	goal := status.Mission.Order.(flagmissions.BreakShields).Goal

	for n := range goal + 1 {
		require.NoError(t, useCase.Execute(t.Context(), flagmissions.Strike{
			Flag: "fr", Owner: "dz", At: at.Add(time.Duration(n+1) * time.Millisecond),
		}))
	}

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: status.Mission.ID.String(), Country: "fr", Points: status.Mission.Points,
		GrantedAt: timestamppb.New(at.Add(time.Duration(goal) * time.Millisecond)),
	}, outbox.Recorded()[0]), "the strike that broke the last shield")
	day, err := store.Day(t.Context(), calendar.DayOf(at), "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(goal+1), day.Snapshot().Strikes)
}
