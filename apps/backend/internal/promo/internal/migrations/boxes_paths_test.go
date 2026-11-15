package migrations_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

// before is the schema as it was before the boxes left the bonuses module.
func before(t *testing.T) fstest.MapFS {
	t.Helper()

	kept := fstest.MapFS{}
	for _, name := range []string{"20261105120000_promo.up.sql", "20261105120000_promo.down.sql"} {
		data, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		kept[name] = &fstest.MapFile{Data: data}
	}
	return kept
}

func TestACardThatNamedABoxValueUnderBonusesNamesItUnderBoxes(t *testing.T) {
	database := server.CreateDatabase(t)
	old := database.OpenSchema(t, "promo", before(t))
	_, err := old.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO backgrounds (background_id, name, media_type, size_bytes, kept_at)
		VALUES ('00000000-0000-0000-0000-000000000001', 'war', 'image/png', 1, now());
		INSERT INTO cards (rule_set, position, background_id, headline, line) VALUES
			('urf', 0, '00000000-0000-0000-0000-000000000001', 'Bombs, spreads, encloses',
				'A bonus box drops every {bonuses.boxes.minInterval} to {bonuses.boxes.maxInterval}.'),
			('urf', 1, '00000000-0000-0000-0000-000000000001', '{bonuses.boxes.maxChargesPerHour} boxes an hour',
				'Gone in {bonuses.offerTtl}, again in {bonuses.missRetry}, newcomers for {bonuses.welcome.within}'),
			('urf', 2, '00000000-0000-0000-0000-000000000001', 'Up to {bonuses.shield.maxPerBox} shields',
				'{clicks.tempo}× faster')`)
	require.NoError(t, err)

	db := database.OpenSchema(t, "promo", migrations.FS)

	found, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT headline, line FROM cards ORDER BY position`)
	require.NoError(t, err)
	defer func() { _ = found.Close() }()
	var words []string
	for found.Next() {
		var headline, line string
		require.NoError(t, found.Scan(&headline, &line))
		words = append(words, headline, line)
	}
	require.NoError(t, found.Err())

	assert.Equal(t, []string{
		"Bombs, spreads, encloses", "A bonus box drops every {boxes.minInterval} to {boxes.maxInterval}.",
		"{boxes.maxBoxesPerHour} boxes an hour", "Gone in {boxes.offerTtl}, again in {boxes.missRetry}, newcomers for {boxes.welcome.within}",
		"Up to {bonuses.shield.maxPerBox} shields", "{clicks.tempo}× faster",
	}, words, "what a box holds stays the bonuses module's")
}
