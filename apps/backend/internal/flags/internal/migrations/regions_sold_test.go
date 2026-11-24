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

func TestEnglandScotlandAndWalesAreSoldUnderTheCodesTheyShippedWithOpenForThirtyDays(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)

	for _, want := range []struct {
		code    catalog.FlagID
		name    catalog.Name
		picture string
		pin     catalog.Pin
	}{
		{"gb-eng", "England", "a5bbe1742db6aa08e5850c063ca9feb11b17c99fbf2546f07ed1f2e57bb44edb", catalog.Pin{Latitude: 51.501941, Longitude: -0.118668}},
		{"gb-sct", "Scotland", "e044e22d0d52a24b10503b26b85c66d0c9223e2ddd07842b65d0c9b00ab0a5f8", catalog.Pin{Latitude: 55.948328, Longitude: -3.219091}},
		{"gb-wls", "Wales", "625234995a5a9c1da4e98805b1f62053d49ee570cc82c79494ae5e6914fb32d5", catalog.Pin{Latitude: 51.499995, Longitude: -3.225008}},
	} {
		flag, err := postgres_flag_store.New(db).Flag(t.Context(), want.code)
		require.NoError(t, err, want.code)

		kept := flag.TakeSnapshot()
		assert.Equal(t, want.name, kept.Name)
		assert.Equal(t, catalog.Open, kept.State, want.code)
		assert.WithinDuration(t, time.Now(), kept.DecidedAt, time.Minute, want.code)
		assert.Equal(t, kept.DecidedAt.Add(30*24*time.Hour), kept.EndsAt, want.code)
		assert.Equal(t, "flags/pictures/"+want.picture+".png", kept.Picture.Key(), want.code)
		assert.Equal(t, [2]uint32{768, 461}, [2]uint32{kept.Picture.Width, kept.Picture.Height}, want.code)
		assert.Equal(t, want.pin, kept.Pin, want.code)
	}
}
