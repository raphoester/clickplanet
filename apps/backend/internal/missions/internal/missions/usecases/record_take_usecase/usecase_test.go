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
	return missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}, p.err
}

type pledged string

func (p pledged) FlagOf(context.Context, missions.AccountID) (string, error) { return string(p), nil }

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
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.NoError(t, useCase.Execute(t.Context(), missions.Take{
		Account: ada, Flag: "fr", Previous: "dz", At: at, Position: 41,
	}))

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	drawn := missions.Draw(calendar.DayOf(at), ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()
	assert.Equal(t, missions.SheetSnapshot{
		Progress:   map[missions.MissionID]uint32{drawn[0].ID: 1, drawn[1].ID: 1, drawn[2].ID: 1},
		Paid:       cpcolls.NewSet[missions.MissionID](),
		TakesBelow: missions.TakeMark{Position: 41, Index: 1},
	}, sheet.Snapshot(), "with no ground big enough, the third mission is the flag's too")
}

func TestASpreadCountsEachTileItTookOnce(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	for range 2 {
		for index := range uint32(7) {
			require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at, Position: 41, Index: index}))
		}
	}

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	drawn := missions.Draw(calendar.DayOf(at), ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()
	assert.Equal(t, uint32(7), sheet.Snapshot().Progress[drawn[0].ID], "one ledger event, seven takes, told twice")
}

func TestAnAccountsMissionsAreDrawnOnceADay(t *testing.T) {
	store := inmemory_mission_store.New()
	planet := &countingPlanet{}
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, planet, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at}))
	}
	require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at.AddDate(0, 0, 1)}))

	assert.Equal(t, 2, planet.asked, "once on each day")
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{err: errors.New("down")}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.Error(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at}))

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(at), ada)
	require.NoError(t, err)
	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestOnlyTheTakeThatFinishesAMissionPaysIt(t *testing.T) {
	store := inmemory_mission_store.New()
	payroll := &recordingPayroll{}
	useCase := record_take_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, payroll)

	drawn := missions.Draw(calendar.DayOf(at), ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()
	goal := drawn[0].Order.(missions.TakeFor).Goal

	for position := range uint64(goal + 1) {
		require.NoError(t, useCase.Execute(t.Context(), missions.Take{Account: ada, Flag: "fr", At: at, Position: position}))
		if position == uint64(goal)-2 {
			require.Empty(t, payroll.paid, "one short of the bank")
		}
	}

	assert.Equal(t, []missions.AccountID{ada}, payroll.paid, "the take that finished the bank paid it, the next nothing")
}
