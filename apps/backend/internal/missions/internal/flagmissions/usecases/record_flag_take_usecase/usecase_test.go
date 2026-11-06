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

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
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
	at    = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	today = days.TestingDayAt(at)
)

type recordingStreams struct {
	woken []missions.AccountID
}

func (s *recordingStreams) Publish(account missions.AccountID) {
	s.woken = append(s.woken, account)
}

func setUp(planet *countingPlanet, told ...days.Day) (*record_flag_take_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox) {
	useCase, store, outbox, _ := setUpWatched(planet, told...)
	return useCase, store, outbox
}

func setUpWatched(
	planet *countingPlanet, told ...days.Day,
) (*record_flag_take_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox, *recordingStreams) {
	store, outbox, calendar := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}, inmemory_day_store.New(append(told, today)...)
	streams := &recordingStreams{}
	return record_flag_take_usecase.New(calendar, flagmissions.NewBriefing(store, calendar, planet, noPledges{}), store,
		cpatomic.FakeTransactioner{}, outbox, streams), store, outbox, streams
}

func take(position uint64) flagmissions.Take {
	return flagmissions.Take{Flag: "fr", Ground: "br", At: at, Position: position}
}

func TestATakeCountsOnItsFlagsDayDrawnOnce(t *testing.T) {
	planet := &countingPlanet{}
	useCase, store, _ := setUp(planet)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))
	require.NoError(t, useCase.Execute(t.Context(), take(2)))

	day, err := store.Day(t.Context(), today, "fr")
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
	_, err = store.KeepDay(t.Context(), today, "fr", almost)
	require.NoError(t, err)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))
	require.NoError(t, useCase.Execute(t.Context(), take(2)))

	require.Len(t, outbox.Recorded(), 2)
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: mission.String(), Country: "fr", Points: 3, GrantedAt: timestamppb.New(at),
	}, outbox.Recorded()[0]))
	assert.True(t, proto.Equal(&missionsv1.FlagMissionDone{
		Flag: "fr",
		Mission: &missionsv1.FlagMission{Id: mission.String(), Points: 3, Done: true, Kind: &missionsv1.FlagMission_TakeIn{
			TakeIn: &missionsv1.TakeIn{CountryId: "br", Goal: 2, Taken: 2},
		}},
		DoneAt: timestamppb.New(at),
	}, outbox.Recorded()[1]), "told in the transaction that gives the points")
}

func TestADayClearedOfItsMissionIsDrawnAgainAndKeepsWhatItCounted(t *testing.T) {
	planet := &countingPlanet{}
	useCase, store, _ := setUp(planet)
	cleared, err := flagmissions.DayOf(flagmissions.DaySnapshot{Takes: 40, TakesBelow: missions.LedgerMark{Position: 40, Index: 1}})
	require.NoError(t, err)
	_, err = store.KeepDay(t.Context(), today, "fr", cleared)
	require.NoError(t, err)

	require.NoError(t, useCase.Execute(t.Context(), take(41)))
	require.NoError(t, useCase.Execute(t.Context(), take(42)))

	assert.Equal(t, 1, planet.asked, "drawn once")
	world, err := planet.World(t.Context())
	require.NoError(t, err)
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, flagmissions.Draw(today, "fr", world, flagmissions.Day{}, flagmissions.Played).Snapshot().Mission, day.Snapshot().Mission)
	assert.Equal(t, uint64(42), day.Snapshot().Takes)
	assert.Equal(t, uint32(2), day.Snapshot().Progress, "only the takes after the draw count on the mission")
}

func TestAMapThatCannotBeReadCountsNothing(t *testing.T) {
	useCase, store, _ := setUp(&countingPlanet{err: errors.New("down")})

	require.Error(t, useCase.Execute(t.Context(), take(1)))

	_, err := store.Day(t.Context(), today, "fr")
	assert.ErrorIs(t, err, flagmissions.ErrNoDay)
}

func TestATakeWhenNoDayRunsCountsNothing(t *testing.T) {
	planet := &countingPlanet{}
	useCase, store, outbox := setUp(planet)
	late := take(1)
	late.At = today.Snapshot().EndsAt

	require.NoError(t, useCase.Execute(t.Context(), late))

	_, err := store.Day(t.Context(), today, "fr")
	require.ErrorIs(t, err, flagmissions.ErrNoDay)
	assert.Zero(t, planet.asked)
	assert.Empty(t, outbox.Recorded())
}

func TestTheGoalIsDrawnFromWhatTheFlagDidTheDayBefore(t *testing.T) {
	yesterday := days.TestingDayAt(at.Add(-24 * time.Hour))
	planet := &countingPlanet{}
	useCase, store, _ := setUp(planet, yesterday)
	busy, err := flagmissions.DayOf(flagmissions.DaySnapshot{Takes: 10_005})
	require.NoError(t, err)
	_, err = store.KeepDay(t.Context(), yesterday, "fr", busy)
	require.NoError(t, err)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))

	world, err := planet.World(t.Context())
	require.NoError(t, err)
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, flagmissions.Draw(today, "fr", world, busy, flagmissions.Played).Snapshot().Mission, day.Snapshot().Mission)
}

func TestASeasonsFirstDayHasTheLeastGoal(t *testing.T) {
	planet := &countingPlanet{}
	useCase, store, _ := setUp(planet)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))

	world, err := planet.World(t.Context())
	require.NoError(t, err)
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, flagmissions.Draw(today, "fr", world, flagmissions.Day{}, flagmissions.Played).Snapshot().Mission, day.Snapshot().Mission)
}

func TestATakeWakesTheStreamsOfTheAccountThatTookIt(t *testing.T) {
	useCase, _, _, streams := setUpWatched(&countingPlanet{})
	ada := missions.AccountID{15: 1}

	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Take{Account: ada, Flag: "fr", Ground: "br", At: at, Position: 1}))
	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Take{Flag: "fr", Ground: "br", At: today.Snapshot().EndsAt}))

	assert.Equal(t, []missions.AccountID{ada}, streams.woken, "a take when no day runs counts nothing and wakes nothing")
}
