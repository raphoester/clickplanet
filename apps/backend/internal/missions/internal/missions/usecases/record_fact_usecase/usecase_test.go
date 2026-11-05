package record_fact_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
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

func (p *recordingPayroll) Pay(_ context.Context, _ days.Day, account missions.AccountID) error {
	p.paid = append(p.paid, account)
	return nil
}

var (
	at       = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
	calendar = inmemory_day_store.New(days.TestingDayAt(at), days.TestingDayAt(at.AddDate(0, 0, 1)))
)

func TestATakeCountsOnTheMissionsOfItsDay(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{
		Account: ada, At: at, Fact: missions.Take{Flag: "fr", Previous: "dz", Position: 41},
	}))

	sheet, err := store.Sheet(t.Context(), days.TestingDayAt(at), ada)
	require.NoError(t, err)
	drawn := missions.Draw(days.TestingDayAt(at), ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()
	assert.Equal(t, missions.SheetSnapshot{
		Progress: map[missions.MissionID]uint32{drawn[0].ID: 1, drawn[1].ID: 1, drawn[2].ID: 1},
		Paid:     cpcolls.NewSet[missions.MissionID](),
		Counted:  missions.Counted{TakesBelow: missions.LedgerMark{Position: 41, Index: 1}},
	}, sheet.Snapshot(), "with no ground big enough, the third mission is the flag's too")
}

func TestASpreadCountsEachTileItTookOnce(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	for range 2 {
		for index := range uint32(7) {
			require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{
				Account: ada, At: at, Fact: missions.Take{Flag: "fr", Position: 41, Index: index},
			}))
		}
	}

	sheet, err := store.Sheet(t.Context(), days.TestingDayAt(at), ada)
	require.NoError(t, err)
	drawn := missions.Draw(days.TestingDayAt(at), ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()
	assert.Equal(t, uint32(7), sheet.Snapshot().Progress[drawn[0].ID], "one ledger event, seven takes, told twice")
}

func TestAnAccountsMissionsAreDrawnOnceADay(t *testing.T) {
	store := inmemory_mission_store.New()
	planet := &countingPlanet{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, planet, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr"}}))
	}
	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at.AddDate(0, 0, 1), Fact: missions.Take{Flag: "fr"}}))

	assert.Equal(t, 2, planet.asked, "once on each day")
}

func TestAFactWhenNoDayRunsCountsNothing(t *testing.T) {
	store, planet := inmemory_mission_store.New(), &countingPlanet{}
	useCase := record_fact_usecase.New(inmemory_day_store.New(), missions.NewBriefing(store, planet, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr"}}))

	sheet, err := store.Sheet(t.Context(), days.TestingDayAt(at), ada)
	require.NoError(t, err)
	assert.Empty(t, sheet.Snapshot().Progress)
	assert.Zero(t, planet.asked, "no missions are drawn")
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{err: errors.New("down")}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{})

	require.Error(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr"}}))

	sheet, err := store.Sheet(t.Context(), days.TestingDayAt(at), ada)
	require.NoError(t, err)
	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestOnlyTheTakeThatFinishesAMissionPaysIt(t *testing.T) {
	store := inmemory_mission_store.New()
	payroll := &recordingPayroll{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, payroll)

	drawn := missions.Draw(days.TestingDayAt(at), ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()
	goal := drawn[0].Order.(missions.TakeFor).Goal

	for position := range uint64(goal + 1) {
		require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{
			Account: ada, At: at, Fact: missions.Take{Flag: "fr", Position: position},
		}))
		if position == uint64(goal)-2 {
			require.Empty(t, payroll.paid, "one short of the bank")
		}
	}

	assert.Equal(t, []missions.AccountID{ada}, payroll.paid, "the take that finished the bank paid it, the next nothing")
}
