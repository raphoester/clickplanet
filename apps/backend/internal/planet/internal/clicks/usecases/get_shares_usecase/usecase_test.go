package get_shares_usecase_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_shares_usecase"
)

func TestTheSharesAreEveryCountrysTilesOverTheWholeMap(t *testing.T) {
	storage := inmemory_tile_storage.New(100, inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), slog.New(slog.DiscardHandler))
	for tile := uint32(1); tile <= 3; tile++ {
		require.NoError(t, storage.Set(t.Context(), tile, "dz"))
	}
	require.NoError(t, storage.Set(t.Context(), 4, "fr"))

	out := get_shares_usecase.New(storage, clicks.NewBoard(100)).Execute(t.Context())

	assert.Equal(t, uint32(100), out.MapTiles)
	assert.ElementsMatch(t, []clicks.Holding{{Country: "dz", Tiles: 3}, {Country: "fr", Tiles: 1}}, out.Holdings)
}
