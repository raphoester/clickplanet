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

type recordingStreams struct {
	woken []missions.AccountID
}

func (s *recordingStreams) Publish(account missions.AccountID) {
	s.woken = append(s.woken, account)
}

type countingTransactioner struct {
	opened int
}

func (c *countingTransactioner) InTx(ctx context.Context, do func(ctx context.Context) error) error {
	c.opened++
	return cpatomic.FakeTransactioner{}.InTx(ctx, do) //nolint:wrapcheck // a fake.
}

var (
	at       = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
	grace    = missions.AccountID{15: 2}
	calendar = inmemory_day_store.New(days.TestingDayAt(at), days.TestingDayAt(at.AddDate(0, 0, 1)))
	world    = missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}
)

func take(account missions.AccountID, at time.Time, position uint64, index uint32) record_fact_usecase.In {
	return record_fact_usecase.In{Account: account, At: at, Fact: missions.Take{Flag: "fr", Position: position, Index: index}}
}

func progressOf(t *testing.T, store *inmemory_mission_store.Store, day days.Day, account missions.AccountID) uint32 {
	t.Helper()

	sheet, err := store.Sheet(t.Context(), day, account)
	require.NoError(t, err)
	return sheet.Snapshot().Progress[missions.Draw(day, account, "fr", world).Snapshot()[0].ID]
}

func TestATakeCountsOnTheMissionsOfItsDay(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{}, &recordingStreams{})

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
		cpatomic.FakeTransactioner{}, &recordingPayroll{}, &recordingStreams{})

	for range 2 {
		for index := range uint32(7) {
			require.NoError(t, useCase.Execute(t.Context(), take(ada, at, 41, index)))
		}
	}

	assert.Equal(t, uint32(7), progressOf(t, store, days.TestingDayAt(at), ada), "one ledger event, seven takes, told twice")
}

func TestABatchToldAgainWholeOrInPartCountsEachTakeOnce(t *testing.T) {
	store := inmemory_mission_store.New()
	transactions := &countingTransactioner{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		transactions, &recordingPayroll{}, &recordingStreams{})
	enclosure := make([]record_fact_usecase.In, 0, 7)
	for index := range uint32(7) {
		enclosure = append(enclosure, take(ada, at, 41, index))
	}

	for range 2 {
		require.NoError(t, useCase.Execute(t.Context(), enclosure...))
	}
	require.NoError(t, useCase.Execute(t.Context(), enclosure[3:]...), "a failed batch is handled again one event at a time")

	assert.Equal(t, uint32(7), progressOf(t, store, days.TestingDayAt(at), ada), "one ledger event, seven takes")
	assert.Equal(t, 3, transactions.opened, "one transaction per account's day of a batch, not one per take")
}

func TestABatchCountsEachAccountsTakesOnItsOwnDayAndWakesEachAccountOnce(t *testing.T) {
	store, planet, streams := inmemory_mission_store.New(), &countingPlanet{}, &recordingStreams{}
	transactions := &countingTransactioner{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, planet, pledged("fr")), store,
		transactions, &recordingPayroll{}, streams)
	tomorrow := at.AddDate(0, 0, 1)

	require.NoError(t, useCase.Execute(t.Context(),
		take(ada, at, 1, 0), take(grace, at, 2, 0), take(ada, at, 3, 0), take(ada, at, 3, 1), take(ada, tomorrow, 4, 0),
		take(grace, at.AddDate(0, 0, 3), 5, 0),
	))

	assert.Equal(t, uint32(3), progressOf(t, store, days.TestingDayAt(at), ada))
	assert.Equal(t, uint32(1), progressOf(t, store, days.TestingDayAt(tomorrow), ada), "a take counts on the day it ran in")
	assert.Equal(t, uint32(1), progressOf(t, store, days.TestingDayAt(at), grace))
	assert.Equal(t, 3, transactions.opened, "one per account and day; a take when no day runs counts nothing")
	assert.Equal(t, 3, planet.asked, "each account's missions drawn once a day")
	assert.Equal(t, []missions.AccountID{ada, grace}, streams.woken, "once per account, in the order they came")
}

func TestAnAccountsMissionsAreDrawnOnceADay(t *testing.T) {
	store := inmemory_mission_store.New()
	planet := &countingPlanet{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, planet, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{}, &recordingStreams{})

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr"}}))
	}
	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at.AddDate(0, 0, 1), Fact: missions.Take{Flag: "fr"}}))

	assert.Equal(t, 2, planet.asked, "once on each day")
}

func TestAFactWhenNoDayRunsCountsNothing(t *testing.T) {
	store, planet := inmemory_mission_store.New(), &countingPlanet{}
	useCase := record_fact_usecase.New(inmemory_day_store.New(), missions.NewBriefing(store, planet, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{}, &recordingStreams{})

	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr"}}))

	sheet, err := store.Sheet(t.Context(), days.TestingDayAt(at), ada)
	require.NoError(t, err)
	assert.Empty(t, sheet.Snapshot().Progress)
	assert.Zero(t, planet.asked, "no missions are drawn")
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	store := inmemory_mission_store.New()
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{err: errors.New("down")}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{}, &recordingStreams{})

	require.Error(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr"}}))

	sheet, err := store.Sheet(t.Context(), days.TestingDayAt(at), ada)
	require.NoError(t, err)
	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestOnlyTheTakeThatFinishesAMissionPaysIt(t *testing.T) {
	store := inmemory_mission_store.New()
	payroll := &recordingPayroll{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, payroll, &recordingStreams{})

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

func TestAMissionFinishedMidBatchIsPaidOnce(t *testing.T) {
	store := inmemory_mission_store.New()
	payroll := &recordingPayroll{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, payroll, &recordingStreams{})
	goal := missions.Draw(days.TestingDayAt(at), ada, "fr", world).Snapshot()[0].Order.(missions.TakeFor).Goal
	batch := make([]record_fact_usecase.In, 0, goal+10)
	for index := range goal + 10 {
		batch = append(batch, take(ada, at, 1, index))
	}

	require.NoError(t, useCase.Execute(t.Context(), batch...))
	require.NoError(t, useCase.Execute(t.Context(), batch...))

	assert.Equal(t, []missions.AccountID{ada}, payroll.paid, "the batch that finished the bank paid it, the same told again nothing")
	assert.Equal(t, goal, progressOf(t, store, days.TestingDayAt(at), ada))
}

func TestACountedFactWakesTheAccountsStreams(t *testing.T) {
	store, streams := inmemory_mission_store.New(), &recordingStreams{}
	useCase := record_fact_usecase.New(calendar, missions.NewBriefing(store, &countingPlanet{}, pledged("fr")), store,
		cpatomic.FakeTransactioner{}, &recordingPayroll{}, streams)

	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at, Fact: missions.Take{Flag: "fr", Position: 1}}))
	require.NoError(t, useCase.Execute(t.Context(), record_fact_usecase.In{Account: ada, At: at.AddDate(0, 0, 2), Fact: missions.Take{Flag: "fr"}}))

	assert.Equal(t, []missions.AccountID{ada}, streams.woken, "a fact when no day runs wakes nothing")
}
