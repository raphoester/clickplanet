package random_flag_ids_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/random_flag_ids"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

func TestIDsAreDistinctAndNoFlagTheGameShipsWithCanTakeOne(t *testing.T) {
	drawn := cpcolls.NewSet[catalog.FlagID]()
	for range 1000 {
		id, err := random_flag_ids.Provider{}.NewID()
		require.NoError(t, err)
		_, err = catalog.ParseFlagID(id.String())
		require.NoError(t, err)
		assert.False(t, cpflags.IsBuiltIn(id.String()))
		assert.False(t, drawn.Contains(id), "drawn twice: %s", id)
		drawn.Add(id)
	}
}
