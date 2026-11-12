package get_tiles_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tiles_handler"
)

type stubQuery struct {
	asked [][]uint32
}

func (s *stubQuery) Tiles(tiles []uint32) *planetv1.GetTilesResponse {
	s.asked = append(s.asked, tiles)
	return &planetv1.GetTilesResponse{Tiles: []*planetv1.MapTile{{TileId: 7, CountryId: "fr"}}}
}

func ask(t *testing.T, query *stubQuery, tiles []uint32) (*connect.Response[planetv1.GetTilesResponse], error) {
	t.Helper()
	return get_tiles_handler.New(clicks.NewBoard(get_tiles_handler.MaxTiles+10), query).GetTiles( //nolint:wrapcheck // the test reads the handler's own error.
		t.Context(), connect.NewRequest(&planetv1.GetTilesRequest{TileIds: tiles}))
}

func TestTheTilesAreTheQuerys(t *testing.T) {
	query := &stubQuery{}

	res, err := ask(t, query, []uint32{7})

	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.GetTilesResponse{Tiles: []*planetv1.MapTile{{TileId: 7, CountryId: "fr"}}}, res.Msg))
	assert.Equal(t, [][]uint32{{7}}, query.asked)
}

func TestATileOffTheMapIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	for _, tile := range []uint32{0, get_tiles_handler.MaxTiles + 11} {
		_, err := ask(t, query, []uint32{7, tile})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "tile %d", tile)
		assert.ErrorIs(t, err, clicks.ErrTileOutOfRange)
	}
	assert.Empty(t, query.asked)
}

func TestMoreTilesThanOneReadAnswersAreInvalid(t *testing.T) {
	query := &stubQuery{}
	tiles := make([]uint32, get_tiles_handler.MaxTiles+1)
	for i := range tiles {
		tiles[i] = uint32(i + 1) //nolint:gosec // a few thousand.
	}

	_, err := ask(t, query, tiles)

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorIs(t, err, get_tiles_handler.ErrTooManyTiles)
	assert.Empty(t, query.asked)
}
