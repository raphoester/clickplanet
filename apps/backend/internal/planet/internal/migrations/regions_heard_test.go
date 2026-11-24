package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog/postgres_catalog_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestEnglandScotlandAndWalesPlayOnTheirPinsTheMomentTheyLeaveTheBuild(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "planet", migrations.FS)
	flags := catalog.NewCatalog(postgres_catalog_store.New(db), catalog.Unlocked)

	for _, flag := range []string{"gb-eng", "gb-sct", "gb-wls"} {
		playable, err := flags.Playable(t.Context(), flag)
		require.NoError(t, err)
		assert.True(t, playable, flag)
	}
	pinned, err := flags.Pinned(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []clicks.Pinned{
		{Flag: "gb-eng", Pin: clicks.Pin{Latitude: 51.501941, Longitude: -0.118668}},
		{Flag: "gb-sct", Pin: clicks.Pin{Latitude: 55.948328, Longitude: -3.219091}},
		{Flag: "gb-wls", Pin: clicks.Pin{Latitude: 51.499995, Longitude: -3.225008}},
	}, pinned)
}
