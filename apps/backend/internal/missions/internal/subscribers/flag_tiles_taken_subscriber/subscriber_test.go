package flag_tiles_taken_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/usecases/record_flag_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/flag_tiles_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type recordingUseCase struct {
	calls [][]flagmissions.Take
}

func (u *recordingUseCase) Execute(_ context.Context, takes ...flagmissions.Take) error {
	u.calls = append(u.calls, takes)
	return nil
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func takes(account, flag string, position uint64, from, to uint32) *planetv1.TilesTaken {
	tiles := make([]*planetv1.TakenTile, 0, to-from)
	for index := from; index < to; index++ {
		tiles = append(tiles, &planetv1.TakenTile{TileId: 40 + index, Index: index, Previous: "dz", Ground: "br"})
	}
	return &planetv1.TilesTaken{AccountId: account, Country: flag, TakenAt: timestamppb.New(at), Position: position, Tiles: tiles}
}

func TestEachTileCountsForTheFlagItWasTakenForWithItsAccountOrNone(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, flag_tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		takes(ada, "fr", 7, 2, 4), takes("", "fr", 8, 0, 1),
	}))

	id, err := missions.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, [][]flagmissions.Take{{
		{Account: id, Flag: "fr", Ground: "br", Position: 7, Index: 2, At: at},
		{Account: id, Flag: "fr", Ground: "br", Position: 7, Index: 3, At: at},
		{Flag: "fr", Ground: "br", Position: 8, At: at},
	}}, useCase.calls, "a moderator's write counts for the flag with no account")
}

func TestEachFlagsTakesOfABatchAreRecordedInOneCall(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, flag_tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		takes(ada, "fr", 1, 0, 1), takes(ada, "de", 2, 0, 1), takes(ada, "fr", 3, 0, 2), takes(ada, "de", 4, 0, 0),
	}))

	require.Len(t, useCase.calls, 2, "one call per flag, and an event with no tile records nothing")
	assert.Len(t, useCase.calls[0], 3)
	assert.Equal(t, "fr", useCase.calls[0][2].Flag)
	assert.Len(t, useCase.calls[1], 1)
	assert.Equal(t, "de", useCase.calls[1][0].Flag)
}

func TestABatchWithTakesWithNoTimeIsRefused(t *testing.T) {
	useCase := &recordingUseCase{}
	noTime := takes(ada, "fr", 2, 0, 1)
	noTime.TakenAt = nil

	err := flag_tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, "fr", 1, 0, 1), noTime})

	require.Error(t, err)
	assert.Empty(t, useCase.calls)
}

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{CountrySizes: []missions.Holding{{Country: "br", Tiles: 9_000}}}, nil
}

type noPledges struct{}

func (noPledges) FlagOf(context.Context, missions.AccountID) (string, error) { return "", nil }

type noStreams struct{}

func (noStreams) Publish(missions.AccountID) {}

func TestAnEnclosureToldInPartsAndToldTwiceCountsEachTileOnce(t *testing.T) {
	store, calendar := inmemory_flag_day_store.New(), inmemory_day_store.New(days.TestingDayAt(at))
	handler := flag_tiles_taken_subscriber.New(record_flag_take_usecase.New(calendar, flagmissions.NewBriefing(store, calendar,
		fixedPlanet{}, noPledges{}, flagmissions.NewFixedSizings(flagmissions.TestingSizing())), store,
		cpatomic.FakeTransactioner{}, &cpatomic.FakeOutbox{}, noStreams{}))
	enclosure := []*planetv1.TilesTaken{takes(ada, "fr", 9, 0, 3), takes(ada, "fr", 9, 3, 7)}

	for range 2 {
		require.NoError(t, handler.HandleBatch(t.Context(), enclosure))
	}
	require.NoError(t, handler.HandleBatch(t.Context(), enclosure[1:]), "a failed batch is handled again one event at a time")

	day, err := store.Day(t.Context(), days.TestingDayAt(at), "fr")
	require.NoError(t, err)
	assert.Equal(t, uint64(7), day.Snapshot().Takes, "one ledger event in two parts, seven takes")
	assert.Equal(t, uint32(7), day.Snapshot().Progress, "each on the ground the mission names")
}
