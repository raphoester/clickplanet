package get_missions_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/inmemory_flag_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{
		Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}, CountrySizes: []missions.Holding{{Country: "br", Tiles: 9_000}},
	}, nil
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
	now = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada = missions.AccountID{15: 1}
	bob = missions.AccountID{15: 2}
)

func setUp(allegiances missions.Allegiances) (*get_missions_usecase.UseCase, *inmemory_mission_store.Store) {
	store := inmemory_mission_store.New()
	calendar := inmemory_day_store.New(days.TestingDayAt(now))
	return get_missions_usecase.New(calendar, cptime.NewFixedClock(now), store,
		missions.NewBriefing(store, fixedPlanet{}, allegiances),
		flagmissions.NewBriefing(inmemory_flag_day_store.New(), calendar, fixedPlanet{}, allegiances)), store
}

func TestTheFirstReadDrawsTheDayForThePledgedFlag(t *testing.T) {
	useCase, _ := setUp(pledges{ada: "fr"})

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	world, err := fixedPlanet{}.World(t.Context())
	require.NoError(t, err)
	assert.Equal(t, missions.Draw(days.TestingDayAt(now), ada, "fr", world).Snapshot(), out.Brief.Snapshot())
	assert.Equal(t, time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC), out.Day.Snapshot().EndsAt)
}

func TestTheDayStaysTheOneDrawnWhateverIsPledgedSince(t *testing.T) {
	pledged := pledges{ada: "fr"}
	useCase, store := setUp(pledged)
	first, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})
	require.NoError(t, err)
	bank := first.Brief.Snapshot()[0].ID
	require.NoError(t, store.ChangeSheet(t.Context(), days.TestingDayAt(now), ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{bank: 7}})
	}))
	pledged[ada] = "de"

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Equal(t, first.Brief.Snapshot(), out.Brief.Snapshot())
	assert.Equal(t, uint32(7), out.Sheet.Snapshot().Progress[bank])
}

func TestWithNoFlagPledgedThereAreNoMissions(t *testing.T) {
	useCase, _ := setUp(pledges{})

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Empty(t, out.Brief.Snapshot())
	_, ok := out.Flag.Status()
	assert.False(t, ok, "and no flag mission")
}

func TestThePlayersOfAFlagShareItsMission(t *testing.T) {
	useCase, _ := setUp(pledges{ada: "fr", bob: "fr"})

	adas, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})
	require.NoError(t, err)
	bobs, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: bob})
	require.NoError(t, err)

	world, err := fixedPlanet{}.World(t.Context())
	require.NoError(t, err)
	assert.Equal(t, flagmissions.Draw(days.TestingDayAt(now), "fr", world, flagmissions.Day{}).Snapshot(), adas.Flag.Snapshot())
	assert.Equal(t, adas.Flag.Snapshot(), bobs.Flag.Snapshot())
	assert.NotEqual(t, adas.Brief.Snapshot(), bobs.Brief.Snapshot(), "each has missions of its own beside it")
}

func TestAPlanetThatFailsIsAnError(t *testing.T) {
	useCase, _ := setUp(failingPledges{})

	_, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	assert.Error(t, err)
}

func TestNoMissionsWhenNoDayRuns(t *testing.T) {
	store, calendar := inmemory_mission_store.New(), inmemory_day_store.New()
	useCase := get_missions_usecase.New(calendar, cptime.NewFixedClock(now), store,
		missions.NewBriefing(store, fixedPlanet{}, pledges{ada: "fr"}),
		flagmissions.NewBriefing(inmemory_flag_day_store.New(), calendar, fixedPlanet{}, pledges{ada: "fr"}))

	out, err := useCase.Execute(t.Context(), get_missions_usecase.In{Account: ada})

	require.NoError(t, err)
	assert.Equal(t, get_missions_usecase.Out{}, out)
}
