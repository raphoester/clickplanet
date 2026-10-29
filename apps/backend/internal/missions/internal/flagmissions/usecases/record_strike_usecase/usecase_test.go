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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
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
	at    = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	today = days.TestingDayAt(at)
)

func setUp() (*record_strike_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox) {
	store, outbox, calendar := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}, inmemory_day_store.New(today)
	return record_strike_usecase.New(calendar, flagmissions.NewBriefing(store, calendar, fixedPlanet{}, noPledges{}), store,
		cpatomic.FakeTransactioner{}, outbox), store, outbox
}

func TestTheShieldsBrokenFinishTheMissionAndGiveTheFlagItsPointsOnce(t *testing.T) {
	useCase, store, outbox := setUp()
	drawn := flagmissions.Draw(today, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}},
		flagmissions.Day{})
	status, ok := drawn.Status()
	require.True(t, ok)
	goal := status.Mission.Order.(flagmissions.BreakShields).Goal

	for n := range goal + 1 {
		require.NoError(t, useCase.Execute(t.Context(), flagmissions.Strike{
			Flag: "fr", Owner: "dz", Position: uint64(n + 1), At: at.Add(time.Duration(n+1) * time.Millisecond),
		}))
	}

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: status.Mission.ID.String(), Country: "fr", Points: status.Mission.Points,
		GrantedAt: timestamppb.New(at.Add(time.Duration(goal) * time.Millisecond)),
	}, outbox.Recorded()[0]), "the strike that broke the last shield")
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(goal+1), day.Snapshot().Strikes)
}

func TestAStrikeWhenNoDayRunsCountsNothing(t *testing.T) {
	useCase, store, outbox := setUp()

	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Strike{Flag: "fr", Owner: "dz", At: today.Snapshot().EndsAt}))

	_, err := store.Day(t.Context(), today, "fr")
	require.ErrorIs(t, err, flagmissions.ErrNoDay)
	assert.Empty(t, outbox.Recorded())
}
