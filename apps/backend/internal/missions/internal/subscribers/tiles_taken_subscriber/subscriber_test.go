package tiles_taken_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/tiles_taken_subscriber"
)

type recordingUseCase struct {
	calls [][]record_fact_usecase.In
}

func (u *recordingUseCase) Execute(_ context.Context, facts ...record_fact_usecase.In) error {
	u.calls = append(u.calls, facts)
	return nil
}

const (
	ada   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	grace = "6f1c2f0e-3a8b-4c5d-9e7f-1a2b3c4d5e6f"
)

var at = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func takes(account string, position uint64, from, to uint32) *planetv1.TilesTaken {
	tiles := make([]*planetv1.TakenTile, 0, to-from)
	for index := from; index < to; index++ {
		tiles = append(tiles, &planetv1.TakenTile{TileId: 40 + index, Index: index, Previous: "dz", Ground: "br"})
	}
	return &planetv1.TilesTaken{AccountId: account, Country: "fr", TakenAt: timestamppb.New(at), Position: position, Tiles: tiles}
}

func idOf(t *testing.T, account string) missions.AccountID {
	t.Helper()

	id, err := missions.AccountIDOf(account)
	require.NoError(t, err)
	return id
}

func TestEachTileReachesTheMissionsWithWhoseTileItWasAndWhoseGround(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, 7, 3, 5)}))

	assert.Equal(t, [][]record_fact_usecase.In{{
		{Account: idOf(t, ada), At: at, Fact: missions.Take{Flag: "fr", Previous: "dz", Ground: "br", Position: 7, Index: 3}},
		{Account: idOf(t, ada), At: at, Fact: missions.Take{Flag: "fr", Previous: "dz", Ground: "br", Position: 7, Index: 4}},
	}}, useCase.calls)
}

func TestEachAccountsTakesOfABatchAreRecordedInOneCallInTheOrderTheyCame(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		takes(ada, 1, 0, 1), takes(grace, 2, 0, 2), takes(ada, 3, 0, 1), {AccountId: grace, TakenAt: timestamppb.New(at), Position: 4},
	}))

	require.Len(t, useCase.calls, 2, "one call per account, and an event with no tile records nothing")
	positions := func(facts []record_fact_usecase.In) []uint64 {
		told := make([]uint64, 0, len(facts))
		for _, in := range facts {
			told = append(told, in.Fact.(missions.Take).Position)
		}
		return told
	}
	assert.Equal(t, []uint64{1, 3}, positions(useCase.calls[0]))
	assert.Equal(t, []uint64{2, 2}, positions(useCase.calls[1]))
}

func TestABatchWithAnEventWithNoAccountOrNoTimeIsRefused(t *testing.T) {
	useCase := &recordingUseCase{}
	subscriber := tiles_taken_subscriber.New(useCase)

	require.ErrorIs(t, subscriber.HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, 0, 0, 1), takes("", 1, 0, 1)}),
		missions.ErrInvalidAccount)
	noTime := takes(ada, 2, 0, 1)
	noTime.TakenAt = nil
	require.Error(t, subscriber.HandleBatch(t.Context(), []*planetv1.TilesTaken{noTime}))
	assert.Empty(t, useCase.calls, "nothing of a refused batch is recorded")
}
