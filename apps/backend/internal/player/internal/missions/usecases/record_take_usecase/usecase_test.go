package record_take_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type countingPlanet struct {
	asked int
	err   error
}

func (p *countingPlanet) World(context.Context) (missions.World, error) {
	p.asked++
	return missions.World{Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}}}, p.err
}

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	at       = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = players.AccountID{15: 1}
)

func TestATakeCountsOnTheMissionsOfItsDay(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}), store)

	require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", Previous: "dz", At: at}))

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	assert.Equal(t, missions.SheetSnapshot{
		Flag: "fr", Taken: map[missions.Slot]uint32{1: 1, 2: 1, 3: 1}, Collected: cpcolls.NewSet[missions.Slot](),
	}, sheet.Snapshot(), "with no ground big enough, the third mission is the flag's too")
}

func TestAFlagsMissionsAreDrawnOnceADay(t *testing.T) {
	store := inmemory_mission_store.New()
	planet := &countingPlanet{}
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, planet), store)

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at}))
	}
	require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at.AddDate(0, 0, 1)}))

	assert.Equal(t, 2, planet.asked, "once on each day")
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{err: errors.New("down")}), store)

	require.Error(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at}))

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	assert.False(t, sheet.Enlisted())
}
