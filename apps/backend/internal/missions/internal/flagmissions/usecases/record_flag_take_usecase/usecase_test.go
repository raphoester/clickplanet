package record_flag_take_usecase_test

import (
	"context"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type countingPlanet struct {
	asked int
}

func (p *countingPlanet) World(context.Context) (missions.World, error) {
	p.asked++
	return missions.World{CountrySizes: []missions.Holding{{Country: "br", Tiles: 9_000}}}, nil
}

type noPledges struct{}

func (noPledges) FlagOf(context.Context, missions.AccountID) (string, error) { return "", nil }

var (
	at       = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	today    = days.TestingDayAt(at)
	standing = flagmissions.TestingStanding(map[string]uint32{"de": 120, "fr": 60})
)

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

func setUp(planet *countingPlanet, told ...days.Day) (*record_flag_take_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox) {
	useCase, store, outbox, _ := setUpWatched(planet, told...)
	return useCase, store, outbox
}

func setUpWatched(
	planet *countingPlanet, told ...days.Day,
) (*record_flag_take_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox, *recordingStreams) {
	return setUpAt(planet, standing, cptime.NewFixedClock(at), told...)
}

func setUpAt(
	planet *countingPlanet, standing flagmissions.Standing, clock cptime.Clock, told ...days.Day,
) (*record_flag_take_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox, *recordingStreams) {
	store, outbox, calendar := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}, inmemory_day_store.New(append(told, today)...)
	streams := &recordingStreams{}
	briefing := flagmissions.NewBriefing(store, calendar, planet, flagmissions.NewFixedStandings(standing), noPledges{},
		flagmissions.NewFixedSizings(flagmissions.TestingSizing()), clock)
	return record_flag_take_usecase.New(calendar, briefing, store, cpatomic.FakeTransactioner{}, outbox, streams), store, outbox, streams
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
	assert.Equal(t, flagmissions.Draw(today, "fr", world, standing, flagmissions.Day{}, flagmissions.TestingSizing()).Snapshot().Mission, day.Snapshot().Mission)
	assert.Equal(t, uint64(42), day.Snapshot().Takes)
	assert.Equal(t, uint32(2), day.Snapshot().Progress, "only the takes after the draw count on the mission")
}

func TestTheMissionIsWorthTheLessTheMoreItsFlagWonInTheSeason(t *testing.T) {
	useCase, store, _ := setUp(&countingPlanet{})

	require.NoError(t, useCase.Execute(t.Context(), take(1), flagmissions.Take{Flag: "de", Ground: "br", At: at, Position: 2}))

	for flag, points := range map[string]uint32{"fr": 6, "de": 3} {
		day, err := store.Day(t.Context(), today, flag)
		require.NoError(t, err)
		assert.Equal(t, points, day.Snapshot().Mission.Points, "%s: Germany leads with 120, France has half", flag)
	}
}

func TestEarlyInTheDayATakeWaitsForTheSeasonsRoundsBeforeItToBeScored(t *testing.T) {
	planet := &countingPlanet{}
	unsettled := flagmissions.StandingOf(flagmissions.StandingSnapshot{Points: map[string]uint32{"fr": 60}})
	useCase, store, outbox, _ := setUpAt(planet, unsettled, cptime.NewFixedClock(today.Snapshot().StartsAt.Add(time.Minute+59*time.Second)))
	early := take(1)
	early.At = today.Snapshot().StartsAt

	require.ErrorIs(t, useCase.Execute(t.Context(), early), flagmissions.ErrUnsettled)

	_, err := store.Day(t.Context(), today, "fr")
	require.ErrorIs(t, err, flagmissions.ErrNoDay, "nothing drawn, nothing counted: the take is told again")
	assert.Zero(t, planet.asked)
	assert.Empty(t, outbox.Recorded())
}

func TestTwoMinutesIntoTheDayItsMissionIsDrawnFromTheTableAsItIs(t *testing.T) {
	unsettled := flagmissions.StandingOf(flagmissions.StandingSnapshot{Points: map[string]uint32{"fr": 60}})
	useCase, store, _, _ := setUpAt(&countingPlanet{}, unsettled, cptime.NewFixedClock(today.Snapshot().StartsAt.Add(2*time.Minute)))

	require.NoError(t, useCase.Execute(t.Context(), take(1)))

	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint32(3), day.Snapshot().Mission.Points, "France leads the table as it is")
	assert.Equal(t, uint64(1), day.Snapshot().Takes)
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
	useCase, store, _ := setUp(&countingPlanet{}, yesterday)
	busy, err := flagmissions.DayOf(flagmissions.DaySnapshot{Takes: 10_005})
	require.NoError(t, err)
	_, err = store.KeepDay(t.Context(), yesterday, "fr", busy)
	require.NoError(t, err)

	require.NoError(t, useCase.Execute(t.Context(), take(1)))

	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, flagmissions.TakeIn{Ground: "br", Goal: 3_000}, day.Snapshot().Mission.Order, "30% of 10,005, rounded down to 10")
}

func TestASeasonsFirstDayHasTheLeastGoal(t *testing.T) {
	useCase, store, _ := setUp(&countingPlanet{})

	require.NoError(t, useCase.Execute(t.Context(), take(1)))

	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, flagmissions.TakeIn{Ground: "br", Goal: 250}, day.Snapshot().Mission.Order)
}

func TestATakeWakesTheStreamsOfTheAccountThatTookIt(t *testing.T) {
	useCase, _, _, streams := setUpWatched(&countingPlanet{})
	ada := missions.AccountID{15: 1}

	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Take{Account: ada, Flag: "fr", Ground: "br", At: at, Position: 1}))
	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Take{Flag: "fr", Ground: "br", At: today.Snapshot().EndsAt}))

	assert.Equal(t, []missions.AccountID{ada}, streams.woken, "a take when no day runs counts nothing and wakes nothing")
}

func TestABatchCountsEachFlagsTakesOnItsDayInOneTransactionEach(t *testing.T) {
	store, outbox, planet := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}, &countingPlanet{}
	calendar, transactions, streams := inmemory_day_store.New(today), &countingTransactioner{}, &recordingStreams{}
	useCase := record_flag_take_usecase.New(calendar, flagmissions.NewBriefing(store, calendar, planet,
		flagmissions.NewFixedStandings(standing), noPledges{}, flagmissions.NewFixedSizings(flagmissions.TestingSizing()),
		cptime.NewFixedClock(at)), store, transactions, outbox, streams)
	ada, grace := missions.AccountID{15: 1}, missions.AccountID{15: 2}
	german := flagmissions.Take{Account: grace, Flag: "de", Ground: "br", At: at, Position: 2}
	late := take(4)
	late.At = today.Snapshot().EndsAt

	require.NoError(t, useCase.Execute(t.Context(),
		flagmissions.Take{Account: ada, Flag: "fr", Ground: "br", At: at, Position: 1}, german,
		flagmissions.Take{Account: ada, Flag: "fr", Ground: "br", At: at, Position: 3}, late,
	))

	french, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), french.Snapshot().Takes, "a take when no day runs counts nothing")
	germans, err := store.Day(t.Context(), today, "de")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), germans.Snapshot().Takes)
	assert.Equal(t, 2, transactions.opened, "one per flag's day, not one per take")
	assert.Equal(t, 2, planet.asked, "each flag's day drawn once")
	assert.Equal(t, []missions.AccountID{ada, grace}, streams.woken, "each account that took, once")
}

func TestAnEnclosureToldInPartsAndToldTwiceCountsEachTakeOnce(t *testing.T) {
	useCase, store, _ := setUp(&countingPlanet{})
	enclosure := make([]flagmissions.Take, 0, 7)
	for index := range uint32(7) {
		enclosure = append(enclosure, flagmissions.Take{Flag: "fr", Ground: "br", At: at, Position: 9, Index: index})
	}

	for range 2 {
		require.NoError(t, useCase.Execute(t.Context(), enclosure[:3]...))
		require.NoError(t, useCase.Execute(t.Context(), enclosure[3:]...))
	}
	require.NoError(t, useCase.Execute(t.Context(), enclosure...))

	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(7), day.Snapshot().Takes)
	assert.Equal(t, missions.LedgerMark{Position: 9, Index: 7}, day.Snapshot().TakesBelow)
}

func TestAMissionFinishedMidBatchGivesItsPointsOnceAtTheTimeOfTheTakeThatFinishedIt(t *testing.T) {
	useCase, store, outbox := setUp(&countingPlanet{})
	mission := flagmissions.MissionID{15: 7}
	fresh, err := flagmissions.DayOf(flagmissions.DaySnapshot{Mission: &flagmissions.MissionSnapshot{
		ID: mission, Order: flagmissions.TakeIn{Ground: "br", Goal: 2}, Points: 3,
	}})
	require.NoError(t, err)
	_, err = store.KeepDay(t.Context(), today, "fr", fresh)
	require.NoError(t, err)
	batch := make([]flagmissions.Take, 0, 4)
	for position := range uint64(4) {
		taken := take(position + 1)
		taken.At = at.Add(time.Duration(position) * time.Second)
		batch = append(batch, taken)
	}

	require.NoError(t, useCase.Execute(t.Context(), batch...))
	require.NoError(t, useCase.Execute(t.Context(), batch...))

	require.Len(t, outbox.Recorded(), 2, "the points and the word, once")
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: mission.String(), Country: "fr", Points: 3, GrantedAt: timestamppb.New(batch[1].At),
	}, outbox.Recorded()[0]))
	assert.Equal(t, batch[1].At, outbox.Recorded()[1].(*missionsv1.FlagMissionDone).GetDoneAt().AsTime(), "told once, with the grant")
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(4), day.Snapshot().Takes)
}
