package record_strike_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/usecases/record_strike_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}, nil
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

type countingTransactioner struct {
	opened int
}

func (c *countingTransactioner) InTx(ctx context.Context, do func(ctx context.Context) error) error {
	c.opened++
	return cpatomic.FakeTransactioner{}.InTx(ctx, do) //nolint:wrapcheck // a fake.
}

func setUp() (*record_strike_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox, *recordingStreams) {
	useCase, store, outbox, streams, _ := setUpCounted()
	return useCase, store, outbox, streams
}

func setUpCounted() (
	*record_strike_usecase.UseCase, *inmemory_flag_day_store.Store, *cpatomic.FakeOutbox, *recordingStreams, *countingTransactioner,
) {
	store, outbox, calendar := inmemory_flag_day_store.New(), &cpatomic.FakeOutbox{}, inmemory_day_store.New(today)
	streams, transactions := &recordingStreams{}, &countingTransactioner{}
	return record_strike_usecase.New(calendar, flagmissions.NewBriefing(store, calendar, fixedPlanet{}, noPledges{}, flagmissions.NewFixedSizings(flagmissions.TestingSizing())), store,
		transactions, outbox, streams), store, outbox, streams, transactions
}

func goalOf(t *testing.T) (flagmissions.MissionSnapshot, uint32) {
	t.Helper()

	drawn := flagmissions.Draw(today, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}},
		flagmissions.Day{}, flagmissions.TestingSizing())
	status, ok := drawn.Status()
	require.True(t, ok)
	return status.Mission, status.Mission.Order.(flagmissions.BreakShields).Goal
}

func TestTheShieldsBrokenFinishTheMissionAndGiveTheFlagItsPointsOnce(t *testing.T) {
	useCase, store, outbox, _ := setUp()
	mission, goal := goalOf(t)

	for n := range goal + 1 {
		require.NoError(t, useCase.Execute(t.Context(), flagmissions.Strike{
			Flag: "fr", Owner: "dz", Position: uint64(n + 1), At: at.Add(time.Duration(n+1) * time.Millisecond),
		}))
	}

	require.Len(t, outbox.Recorded(), 2)
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: mission.ID.String(), Country: "fr", Points: mission.Points,
		GrantedAt: timestamppb.New(at.Add(time.Duration(goal) * time.Millisecond)),
	}, outbox.Recorded()[0]), "the strike that broke the last shield")
	assert.True(t, proto.Equal(&missionsv1.FlagMissionDone{
		Flag: "fr",
		Mission: &missionsv1.FlagMission{
			Id: mission.ID.String(), Points: mission.Points, Done: true,
			Kind: &missionsv1.FlagMission_BreakShields{BreakShields: &missionsv1.BreakShields{CountryId: "dz", Goal: goal, Broken: goal}},
		},
		DoneAt: timestamppb.New(at.Add(time.Duration(goal) * time.Millisecond)),
	}, outbox.Recorded()[1]), "told once, with the grant")
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(goal+1), day.Snapshot().Strikes)
}

func TestABombsStrikesInPartsFinishTheMissionMidBatchOnceInOneTransactionPerPart(t *testing.T) {
	useCase, store, outbox, _, transactions := setUpCounted()
	mission, goal := goalOf(t)
	bomb := make([]flagmissions.Strike, 0, goal+10)
	for index := range goal + 10 {
		bomb = append(bomb, flagmissions.Strike{Flag: "fr", Owner: "dz", Position: 9, Index: index, At: at})
	}
	parts := [][]flagmissions.Strike{bomb[:goal/2], bomb[goal/2:]}

	for range 2 {
		for _, part := range parts {
			require.NoError(t, useCase.Execute(t.Context(), part...))
		}
	}

	require.Len(t, outbox.Recorded(), 2, "the points and the word, once")
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: mission.ID.String(), Country: "fr", Points: mission.Points, GrantedAt: timestamppb.New(at),
	}, outbox.Recorded()[0]))
	day, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(goal+10), day.Snapshot().Strikes, "each shield once")
	assert.Equal(t, 4, transactions.opened)
}

func TestABatchCountsEachFlagsStrikesOnItsDayAndWakesEachAccountOnce(t *testing.T) {
	useCase, store, _, streams, transactions := setUpCounted()
	ada, grace := missions.AccountID{15: 1}, missions.AccountID{15: 2}

	require.NoError(t, useCase.Execute(t.Context(),
		flagmissions.Strike{Account: ada, Flag: "fr", Owner: "dz", Position: 1, At: at},
		flagmissions.Strike{Account: grace, Flag: "de", Owner: "dz", Position: 2, At: at},
		flagmissions.Strike{Account: grace, Flag: "fr", Owner: "dz", Position: 3, At: at},
		flagmissions.Strike{Account: ada, Flag: "fr", Owner: "dz", Position: 4, At: at},
	))

	french, err := store.Day(t.Context(), today, "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(3), french.Snapshot().Strikes)
	germans, err := store.Day(t.Context(), today, "de")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), germans.Snapshot().Strikes)
	assert.Equal(t, 2, transactions.opened, "one per flag's day")
	assert.Equal(t, []missions.AccountID{ada, grace}, streams.woken, "each account once, whatever flag it broke shields for")
}

func TestAStrikeWhenNoDayRunsCountsNothing(t *testing.T) {
	useCase, store, outbox, _ := setUp()

	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Strike{Flag: "fr", Owner: "dz", At: today.Snapshot().EndsAt}))

	_, err := store.Day(t.Context(), today, "fr")
	require.ErrorIs(t, err, flagmissions.ErrNoDay)
	assert.Empty(t, outbox.Recorded())
}

func TestAShieldBrokenWakesTheStreamsOfTheAccountThatBrokeIt(t *testing.T) {
	useCase, _, _, streams := setUp()
	ada := missions.AccountID{15: 1}

	require.NoError(t, useCase.Execute(t.Context(), flagmissions.Strike{Account: ada, Flag: "fr", Owner: "dz", Position: 1, At: at}))

	assert.Equal(t, []missions.AccountID{ada}, streams.woken)
}
