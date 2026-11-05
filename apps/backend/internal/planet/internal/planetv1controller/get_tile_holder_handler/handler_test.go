package get_tile_holder_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tile_holder_handler"
)

type stubQuery struct {
	answer *planetv1.GetTileHolderResponse
	asked  []uint32
}

func (s *stubQuery) TileHolder(_ context.Context, tile uint32) (*planetv1.GetTileHolderResponse, error) {
	s.asked = append(s.asked, tile)
	return s.answer, nil
}

func ask(t *testing.T, query *stubQuery, tile uint32) (*connect.Response[planetv1.GetTileHolderResponse], error) {
	t.Helper()

	return get_tile_holder_handler.New(clicks.NewBoard(10), query).GetTileHolder(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetTileHolderRequest{TileId: tile}))
}

func TestTheTilesHolderIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetTileHolderResponse{Holder: &planetv1.TileHolder{
		AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", CountryId: "fr", TakenAtUnixMs: 1_761_393_600_000,
	}}}

	res, err := ask(t, query, 7)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, []uint32{7}, query.asked)
}

func TestATileOffTheMapIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	for _, tile := range []uint32{0, 11} {
		_, err := ask(t, query, tile)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "tile %d", tile)
	}
	assert.Empty(t, query.asked)
}
