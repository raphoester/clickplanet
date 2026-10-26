package get_missions_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}}}, nil
}

type pledges map[missions.AccountID]string

func (p pledges) FlagOf(_ context.Context, account missions.AccountID) (string, error) {
	return p[account], nil
}

type failingPledges struct{}

func (failingPledges) FlagOf(context.Context, missions.AccountID) (string, error) {
	return "", errors.New("planet is down")
}

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	now      = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
)

func setUp(allegiances get_missions_usecase.Allegiances) (*get_missions_usecase.UseCase, *inmemory_mission_store.Store) {
	store := inmemory_mission_store.New()
	return get_missions_usecase.New(calendar, cptime.NewFixedClock(now), store,
		missions.NewBriefing(store, fixedPlanet{}), allegiances), store
}

func bankOf(flag string) missions.MissionID {
	return missions.Draw(calendar.DayOf(now), flag, missions.World{Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}}}).Snapshot()[0].ID
}

func enlist(t *testing.T, store *inmemory_mission_store.Store, flag string) {
	t.Helper()

	require.NoError(t, store.ChangeSheet(t.Context(), calendar.DayOf(now), ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Flag: flag, Progress: map[missions.MissionID]uint32{bankOf(flag): 7}})
	}))
}

func TestBeforeTheDaysFirstTakeTheMissionsAreThePledgedFlags(t *testing.T) {
	useCase, _ := setUp(pledges{ada: "fr"})

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Equal(t, "fr", out.Flag)
	assert.False(t, out.Sheet.Enlisted())
	assert.Equal(t, missions.Draw(calendar.DayOf(now), "fr", missions.World{
		Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}},
	}).Snapshot(), out.Brief.Snapshot())
	assert.Equal(t, time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC), out.Day.EndsAt())
}

func TestOnceEnlistedTheMissionsAreTheDaysFlagsWhateverIsPledgedSince(t *testing.T) {
	useCase, store := setUp(pledges{ada: "de"})
	enlist(t, store, "fr")

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Equal(t, "fr", out.Flag)
	assert.True(t, out.Sheet.Enlisted())
	assert.Equal(t, uint32(7), out.Sheet.Snapshot().Progress[bankOf("fr")])
	bank, ok := out.Brief.Snapshot()[0].Order.(missions.TakeFor)
	require.True(t, ok)
	assert.Equal(t, "fr", bank.Flag)
}

func TestWithNoFlagPledgedThereAreNoMissions(t *testing.T) {
	useCase, _ := setUp(pledges{})

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Empty(t, out.Brief.Snapshot())
}

func TestAPlanetThatFailsIsAnError(t *testing.T) {
	useCase, _ := setUp(failingPledges{})

	_, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	assert.Error(t, err)
}
