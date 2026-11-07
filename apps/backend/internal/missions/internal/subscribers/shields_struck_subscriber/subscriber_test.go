package shields_struck_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/inmemory_flag_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/usecases/record_strike_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/shields_struck_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type recordingUseCase struct {
	calls [][]flagmissions.Strike
}

func (u *recordingUseCase) Execute(_ context.Context, strikes ...flagmissions.Strike) error {
	u.calls = append(u.calls, strikes)
	return nil
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func struck(account, flag string, position uint64, from, to uint32) *planetv1.ShieldsStruck {
	tiles := make([]*planetv1.StruckTile, 0, to-from)
	for index := from; index < to; index++ {
		tiles = append(tiles, &planetv1.StruckTile{TileId: 40 + index, Index: index, Owner: "dz"})
	}
	return &planetv1.ShieldsStruck{AccountId: account, Country: flag, StruckAt: timestamppb.New(at), Position: position, Tiles: tiles}
}

func TestEachShieldStruckIsAShieldBrokenOfItsOwnerAtItsPlaceInTheLedger(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, shields_struck_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.ShieldsStruck{
		struck(ada, "fr", 7, 2, 4), struck("", "fr", 8, 0, 1),
	}))

	id, err := missions.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, [][]flagmissions.Strike{{
		{Account: id, Flag: "fr", Owner: "dz", Position: 7, Index: 2, At: at},
		{Account: id, Flag: "fr", Owner: "dz", Position: 7, Index: 3, At: at},
		{Flag: "fr", Owner: "dz", Position: 8, At: at},
	}}, useCase.calls, "a strike with no account counts for the flag")
}

func TestEachFlagsStrikesOfABatchAreRecordedInOneCall(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, shields_struck_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.ShieldsStruck{
		struck(ada, "fr", 1, 0, 1), struck(ada, "de", 2, 0, 1), struck(ada, "fr", 3, 0, 2), struck(ada, "de", 4, 0, 0),
	}))

	require.Len(t, useCase.calls, 2, "one call per flag, and an event with no shield records nothing")
	assert.Len(t, useCase.calls[0], 3)
	assert.Equal(t, "fr", useCase.calls[0][2].Flag)
	assert.Len(t, useCase.calls[1], 1)
	assert.Equal(t, "de", useCase.calls[1][0].Flag)
}

func TestABatchWithStrikesWithNoTimeIsRefused(t *testing.T) {
	useCase := &recordingUseCase{}
	noTime := struck(ada, "fr", 2, 0, 1)
	noTime.StruckAt = nil

	err := shields_struck_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.ShieldsStruck{struck(ada, "fr", 1, 0, 1), noTime})

	require.Error(t, err)
	assert.Empty(t, useCase.calls)
}

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}, nil
}

type noPledges struct{}

func (noPledges) FlagOf(context.Context, missions.AccountID) (string, error) { return "", nil }

type noStreams struct{}

func (noStreams) Publish(missions.AccountID) {}

func TestABombsStrikesToldInPartsAndToldTwiceCountEachShieldOnce(t *testing.T) {
	store, calendar := inmemory_flag_day_store.New(), inmemory_day_store.New(days.TestingDayAt(at))
	handler := shields_struck_subscriber.New(record_strike_usecase.New(calendar, flagmissions.NewBriefing(store, calendar,
		fixedPlanet{}, noPledges{}, flagmissions.NewFixedSizings(flagmissions.TestingSizing())), store,
		cpatomic.FakeTransactioner{}, &cpatomic.FakeOutbox{}, noStreams{}))
	bomb := []*planetv1.ShieldsStruck{struck(ada, "fr", 9, 0, 3), struck(ada, "fr", 9, 3, 7)}

	for range 2 {
		require.NoError(t, handler.HandleBatch(t.Context(), bomb))
	}
	require.NoError(t, handler.HandleBatch(t.Context(), bomb[1:]), "a failed batch is handled again one event at a time")

	day, err := store.Day(t.Context(), days.TestingDayAt(at), "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(7), day.Snapshot().Strikes, "one ledger event in two parts, seven shields")
	assert.Equal(t, uint32(7), day.Snapshot().Progress, "each of the rival the mission names")
}
