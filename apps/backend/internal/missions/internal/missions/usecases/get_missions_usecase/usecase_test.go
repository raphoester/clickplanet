package get_missions_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}}}, nil
}

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	now      = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
)

func setUp() (*get_missions_usecase.UseCase, *inmemory_mission_store.Store) {
	store := inmemory_mission_store.New()
	return get_missions_usecase.New(calendar, cptime.NewFixedClock(now), store,
		missions.NewBriefing(store, fixedPlanet{}), cpcountries.New()), store
}

func enlist(t *testing.T, store *inmemory_mission_store.Store, flag string) {
	t.Helper()

	require.NoError(t, store.ChangeSheet(t.Context(), calendar.DayOf(now), ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Flag: flag, Taken: map[missions.Slot]uint32{1: 7}})
	}))
}

func TestBeforeTheDaysFirstTakeTheMissionsAreTheAskedFlags(t *testing.T) {
	useCase, _ := setUp()

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada, Country: "fr"})

	require.NoError(t, err)
	assert.Equal(t, "fr", out.Flag)
	assert.False(t, out.Sheet.Enlisted())
	assert.Equal(t, missions.Draw(calendar.DayOf(now), "fr", missions.World{
		Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}},
	}).Snapshot(), out.Brief.Snapshot())
	assert.Equal(t, time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC), out.Day.EndsAt())
}

func TestOnceEnlistedTheMissionsAreTheDaysFlagsWhateverIsAsked(t *testing.T) {
	useCase, store := setUp()
	enlist(t, store, "fr")

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada, Country: "de"})

	require.NoError(t, err)
	assert.Equal(t, "fr", out.Flag)
	assert.True(t, out.Sheet.Enlisted())
	assert.Equal(t, uint32(7), out.Sheet.Snapshot().Taken[1])
	assert.Equal(t, "fr", out.Brief.Snapshot()[0].Target)
}

func TestWithNoFlagThereAreNoMissions(t *testing.T) {
	useCase, _ := setUp()

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Empty(t, out.Brief.Snapshot())
}

func TestACountryThatIsNoneIsRefused(t *testing.T) {
	useCase, _ := setUp()

	_, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada, Country: "zz"})

	assert.ErrorIs(t, err, get_missions_usecase.ErrUnknownCountry)
}
