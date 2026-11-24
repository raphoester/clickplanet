package migrations_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestBrittanyIsSoldUnderTheCodeItShippedWithOpenForThirtyDays(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)

	brittany, err := postgres_flag_store.New(db).Flag(t.Context(), "xb")
	require.NoError(t, err)

	kept := brittany.TakeSnapshot()
	assert.Equal(t, catalog.Name("Brittany"), kept.Name)
	assert.Equal(t, catalog.Open, kept.State)
	assert.WithinDuration(t, time.Now(), kept.DecidedAt, time.Minute)
	assert.Equal(t, kept.DecidedAt.Add(30*24*time.Hour), kept.EndsAt)
	assert.Equal(t, "flags/pictures/ed47508c67e67af2c0711725441736fbe62b614fb9b9c587a05efc51023c3eb6.png", kept.Picture.Key())
	assert.Equal(t, [2]uint32{768, 512}, [2]uint32{kept.Picture.Width, kept.Picture.Height})
}
