package collect_mission_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/usecases/collect_mission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Holding{{Country: "dz", Tiles: 9_000}}}, nil
}

type recordingCharger struct {
	granted []missions.Reward
	kept    uint32
	err     error
}

func (c *recordingCharger) Grant(_ context.Context, _ players.AccountID, reward missions.Reward) (uint32, error) {
	c.granted = append(c.granted, reward)
	return c.kept, c.err
}

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	now      = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = players.AccountID{15: 1}
)

func setUp(charger *recordingCharger) (*collect_mission_usecase.UseCase, *inmemory_mission_store.Store) {
	store := inmemory_mission_store.New()
	return collect_mission_usecase.New(calendar, cptime.NewFixedClock(now), store,
		missions.NewBriefing(store, fixedPlanet{}), charger), store
}

func took(t *testing.T, store *inmemory_mission_store.Store, taken uint32) {
	t.Helper()

	require.NoError(t, store.ChangeSheet(t.Context(), calendar.DayOf(now), ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Flag: "fr", Taken: map[missions.Slot]uint32{1: taken}})
	}))
}

func collect(t *testing.T, useCase *collect_mission_usecase.UseCase, slot uint32) (missions.Reward, error) {
	t.Helper()

	out, err := useCase.Execute(t.Context(), collect_mission_usecase.In{Account: ada, Linked: true, Slot: slot})
	return out.Reward, err
}

func collected(t *testing.T, store *inmemory_mission_store.Store) *cpcolls.Set[missions.Slot] {
	t.Helper()

	sheet, err := store.Sheet(t.Context(), calendar.DayOf(now), ada)
	require.NoError(t, err)
	return sheet.Snapshot().Collected
}

func TestADoneMissionPaysWhatFit(t *testing.T) {
	charger := &recordingCharger{kept: 7}
	useCase, store := setUp(charger)
	took(t, store, 60)

	reward, err := collect(t, useCase, 1)

	require.NoError(t, err)
	assert.Equal(t, []missions.Reward{{Kind: missions.RewardShields, Amount: 10}}, charger.granted)
	assert.Equal(t, missions.Reward{Kind: missions.RewardShields, Amount: 7}, reward, "a stack that had room for 7")
	assert.Equal(t, cpcolls.NewSet[missions.Slot](1), collected(t, store))
}

func TestAGuestIsPaidNothing(t *testing.T) {
	charger := &recordingCharger{kept: 10}
	useCase, store := setUp(charger)
	took(t, store, 60)

	_, err := useCase.Execute(t.Context(), collect_mission_usecase.In{Account: ada, Slot: 1})

	require.ErrorIs(t, err, collect_mission_usecase.ErrNotLinked)
	assert.Empty(t, charger.granted)
}

func TestAMissionNotDoneIsNotPaid(t *testing.T) {
	for name, taken := range map[string]uint32{"no take today": 0, "59 of 60": 59} {
		charger := &recordingCharger{kept: 10}
		useCase, store := setUp(charger)
		if taken > 0 {
			took(t, store, taken)
		}

		_, err := collect(t, useCase, 1)

		require.ErrorIs(t, err, missions.ErrNotDone, name)
		assert.Empty(t, charger.granted, name)
	}
}

func TestAMissionIsPaidOnce(t *testing.T) {
	charger := &recordingCharger{kept: 10}
	useCase, store := setUp(charger)
	took(t, store, 60)
	_, err := collect(t, useCase, 1)
	require.NoError(t, err)

	_, err = collect(t, useCase, 1)

	require.ErrorIs(t, err, missions.ErrCollected)
	assert.Len(t, charger.granted, 1)
}

func TestAFullStackKeepsTheMissionToCollectLater(t *testing.T) {
	charger := &recordingCharger{kept: 0}
	useCase, store := setUp(charger)
	took(t, store, 60)

	_, err := collect(t, useCase, 1)
	require.ErrorIs(t, err, collect_mission_usecase.ErrNoRoom)
	assert.True(t, collected(t, store).Empty())

	charger.kept = 10
	_, err = collect(t, useCase, 1)
	assert.NoError(t, err)
}

func TestAPlanetThatFailsKeepsTheMissionToCollectLater(t *testing.T) {
	useCase, store := setUp(&recordingCharger{err: errors.New("down")})
	took(t, store, 60)

	_, err := collect(t, useCase, 1)

	require.Error(t, err)
	assert.True(t, collected(t, store).Empty())
}

func TestASlotNoMissionHasIsRefused(t *testing.T) {
	useCase, store := setUp(&recordingCharger{kept: 1})
	took(t, store, 60)

	_, err := collect(t, useCase, 4)

	assert.ErrorIs(t, err, missions.ErrNoSlot)
}
