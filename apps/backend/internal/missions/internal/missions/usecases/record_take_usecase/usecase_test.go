package record_take_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
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

type recordingPayroll struct {
	paid []missions.AccountID
}

func (p *recordingPayroll) Pay(_ context.Context, _ missions.Day, account missions.AccountID) error {
	p.paid = append(p.paid, account)
	return nil
}

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	at       = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
)

func TestATakeCountsOnTheMissionsOfItsDay(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.NoError(t, useCase.Execute(t.Context(), missions.Take{
		Account: ada, Flag: "fr", Previous: "dz", At: at, Position: 41,
	}))

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	drawn := missions.Draw(calendar.DayOf(at), "fr", missions.World{Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}}}).Snapshot()
	assert.Equal(t, missions.SheetSnapshot{
		Flag: "fr", Progress: map[missions.MissionID]uint32{drawn[0].ID: 1, drawn[1].ID: 1, drawn[2].ID: 1},
		Paid:       cpcolls.NewSet[missions.MissionID](),
		TakesBelow: 42,
	}, sheet.Snapshot(), "with no ground big enough, the third mission is the flag's too")
}

func TestAFlagsMissionsAreDrawnOnceADay(t *testing.T) {
	store := inmemory_mission_store.New()
	planet := &countingPlanet{}
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, planet), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at}))
	}
	require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at.AddDate(0, 0, 1)}))

	assert.Equal(t, 2, planet.asked, "once on each day")
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{err: errors.New("down")}), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.Error(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at}))

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	assert.False(t, sheet.Enlisted())
}

func TestOnlyTheTakeThatFinishesAMissionPaysIt(t *testing.T) {
	store := inmemory_mission_store.New()
	payroll := &recordingPayroll{}
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}), store,
		cpatomic.FakeTransactioner{}, payroll)

	for position := range uint64(61) {
		require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at, Position: position}))
		if position == 58 {
			require.Empty(t, payroll.paid, "59 of 60")
		}
	}

	assert.Equal(t, []missions.AccountID{ada}, payroll.paid, "the 60th take finished one bank, the 61st nothing")
}
