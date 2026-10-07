package get_replay_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_replay_handler"
)

var at = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type stubUseCase struct {
	in  replay_usecase.In
	out replay_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in replay_usecase.In) (replay_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func TestTheWindowReachesTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := get_replay_handler.New(useCase).GetReplay(t.Context(), connect.NewRequest(&planetv1.GetReplayRequest{
		Since: timestamppb.New(at), Until: timestamppb.New(at.Add(time.Hour)),
	}))
	require.NoError(t, err)

	assert.Equal(t, replay_usecase.In{Since: at, Until: at.Add(time.Hour)}, useCase.in)
}

func TestAnUnsetUntilIsLeftForTheUseCaseToFill(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := get_replay_handler.New(useCase).GetReplay(t.Context(), connect.NewRequest(&planetv1.GetReplayRequest{
		Since: timestamppb.New(at),
	}))
	require.NoError(t, err)

	assert.True(t, useCase.in.Until.IsZero())
}

func TestAnInvalidWindowIsTheCallersMistake(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("wrapped: %w", replay_usecase.ErrInvalidWindow)}

	_, err := get_replay_handler.New(useCase).GetReplay(t.Context(), connect.NewRequest(&planetv1.GetReplayRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestEachSceneGoesOutAsTheLiveStreamSentIt(t *testing.T) {
	blast := clicks.Blast{
		Tile: 5, CountryID: "fr", Radius: 0.03, Cleared: []uint32{5, 6}, Struck: []uint32{7},
		Point: clicks.Vec3{X: 1},
	}
	useCase := &stubUseCase{out: replay_usecase.Out{
		Since: at, Until: at.Add(time.Hour),
		Opening: clicks.DenseBatch{
			Start: 1, Codes: []string{"", "de"}, Tiles: []byte{1, 0},
			Shields: []clicks.TileShields{{Tile: 1, Shields: 2}},
		},
		Scenes: []ledger.Scene{
			{At: at, Change: &ledger.Change{
				TileUpdate: clicks.TileUpdate{Tile: 1, Value: "fr", Previous: "de", Clicked: true, Shields: 1}, Was: 2,
			}},
			{At: at.Add(time.Second), Blast: &blast},
			{At: at.Add(2 * time.Second), Spread: &ledger.Bonus{Tile: 1, Country: "fr", Tiles: []uint32{2, 3}}},
			{At: at.Add(3 * time.Second), Enclosure: &ledger.Bonus{Tile: 9, Country: "fr", Tiles: []uint32{4}}},
		},
	}}

	res, err := get_replay_handler.New(useCase).GetReplay(t.Context(), connect.NewRequest(&planetv1.GetReplayRequest{
		Since: timestamppb.New(at),
	}))
	require.NoError(t, err)

	assert.Equal(t, at, res.Msg.GetSince().AsTime())
	assert.Equal(t, at.Add(time.Hour), res.Msg.GetUntil().AsTime())

	opening := res.Msg.GetOpening()
	assert.Equal(t, uint32(1), opening.GetStartTileId())
	assert.Equal(t, []string{"", "de"}, opening.GetCodes())
	assert.Equal(t, []byte{1, 0}, opening.GetTiles())
	require.Len(t, opening.GetShields(), 1)
	assert.Equal(t, uint32(2), opening.GetShields()[0].GetShields())

	events := res.Msg.GetEvents()
	require.Len(t, events, 4)
	assert.Equal(t, at, events[0].GetAt().AsTime())

	update := events[0].GetEvent().GetTileUpdate()
	assert.Equal(t, uint32(1), update.GetTileId())
	assert.Equal(t, "fr", update.GetCountryId())
	assert.Equal(t, "de", update.GetPreviousCountryId())
	assert.True(t, update.GetClicked())
	assert.Equal(t, uint32(1), update.GetShields())

	bomb := events[1].GetEvent().GetBombDropped()
	assert.Equal(t, uint32(5), bomb.GetTileId())
	assert.Equal(t, []uint32{5, 6}, bomb.GetClearedTileIds())
	assert.Equal(t, []uint32{7}, bomb.GetStruckTileIds())
	assert.InDelta(t, 1.0, bomb.GetPoint().GetX(), 1e-9)

	spread := events[2].GetEvent().GetTilesSpread()
	assert.Equal(t, uint32(1), spread.GetTileId())
	assert.Equal(t, []uint32{2, 3}, spread.GetSpreadTileIds())

	enclosure := events[3].GetEvent().GetTilesEnclosed()
	assert.Equal(t, uint32(9), enclosure.GetClosingTileId())
	assert.Equal(t, []uint32{4}, enclosure.GetFilledTileIds())
	assert.Empty(t, enclosure.GetWallTileIds(), "the ledger keeps no wall")
}
