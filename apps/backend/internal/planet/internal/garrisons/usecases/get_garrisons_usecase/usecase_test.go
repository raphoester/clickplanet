package get_garrisons_usecase_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/inmemory_garrison_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/usecases/get_garrisons_usecase"
)

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) { return o[tile], true }

func TestOnlyTheGarrisonsStandingForTheirTilesOwnerAreListed(t *testing.T) {
	held := inmemory_garrison_storage.New(inmemory_garrison_storage.Config{}, 10,
		inmemory_garrison_storage.NewMemoryPersistence(
			garrisons.Garrison{Tile: 3, Country: "fr", Defenders: 2},
			garrisons.Garrison{Tile: 4, Country: "fr", Defenders: 5},
			garrisons.Garrison{Tile: 5, Country: "de", Defenders: 1},
		), slog.New(slog.DiscardHandler))
	require.NoError(t, held.Load(t.Context()))

	listed := get_garrisons_usecase.New(held, owners{3: "fr", 4: "de", 5: ""}).Execute(t.Context())

	assert.Equal(t, []garrisons.Garrison{{Tile: 3, Country: "fr", Defenders: 2}}, listed,
		"a flag that lost the tile, by a bomb or an operator, has nobody left on it")
}
