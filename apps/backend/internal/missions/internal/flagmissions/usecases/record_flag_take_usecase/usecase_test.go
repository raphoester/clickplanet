package record_flag_take_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/inmemory_flag_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/usecases/record_flag_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type countingPlanet struct {
	asked int
	err   error
}

func (p *countingPlanet) World(context.Context) (missions.World, error) {
	p.asked++
	return missions.World{CountrySizes: []missions.Holding{{Country: "br", Tiles: 9_000}}}, p.err
}

type noPledges struct{}

func (noPledges) FlagOf(context.Context, missions.AccountID) (string, error) { return "", nil }

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	at       = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
)

func setUp(planet *countingPlanet) (*record_flag_take_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox) {
	store, outbox := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}
	return record_flag_take_usecase.New(calendar, flagmissions.NewBriefing(store, planet, noPledges{}), store,
		cpatomic.FakeTransactioner{}, outbox), store, outbox
}

func take(position uint64) missions.Take {
	return missions.Take{Account: ada, Flag: "fr", Ground: "br", At: at, Position: position}
}

func TestATakeCountsOnItsFlagsDayDrawnOnce(t *testing.T) {
	planet := &countingPlanet{}
	useCase, store, _ := setUp(planet)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))
	require.NoError(t, useCase.Execute(t.Context(), take(2)))

	day, err := store.Day(t.Context(), calendar.DayOf(at), "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), day.Snapshot().Takes)
	assert.Equal(t, 1, planet.asked)
}

func TestTheTakeThatFinishesTheMissionGivesTheFlagItsPointsOnce(t *testing.T) {
	useCase, store, outbox := setUp(&countingPlanet{})
	mission := flagmissions.MissionID{15: 7}
	almost, err := flagmissions.DayOf(flagmissions.DaySnapshot{Progress: 1, Mission: &flagmissions.MissionSnapshot{
		ID: mission, Order: flagmissions.TakeIn{Ground: "br", Goal: 2}, Points: 3,
	}})
	require.NoError(t, err)
	_, err = store.KeepDay(t.Context(), calendar.DayOf(at), "fr", almost)
	require.NoError(t, err)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))
	require.NoError(t, useCase.Execute(t.Context(), take(2)))

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: mission.String(), Country: "fr", Points: 3, GrantedAt: timestamppb.New(at),
	}, outbox.Recorded()[0]))
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	useCase, store, _ := setUp(&countingPlanet{err: errors.New("down")})

	require.Error(t, useCase.Execute(t.Context(), take(1)))

	_, err := store.Day(t.Context(), calendar.DayOf(at), "fr")
	assert.ErrorIs(t, err, flagmissions.ErrNoDay)
}
