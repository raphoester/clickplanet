package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog/postgres_catalog_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestBrittanyPlaysOnTheMomentItLeavesTheBuild(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "planet", migrations.FS)

	playable, err := catalog.NewCatalog(postgres_catalog_store.New(db), catalog.Unlocked).Playable(t.Context(), "xb")
	require.NoError(t, err)
	assert.True(t, playable)
}
