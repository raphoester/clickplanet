package feed_start_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_feed_start_handler/feed_start_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheStartIsWhatTheMigrationKept(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	query := feed_start_query.NewPostgresQuery(db)

	fresh, err := query.Start(t.Context())
	require.NoError(t, err)
	assert.Zero(t, fresh.GetPosition(), "a ledger that never took a tile starts the feed at its first take")

	_, err = db.ExecContext(t.Context(), `UPDATE ledger_feed SET start = 42`)
	require.NoError(t, err)

	kept, err := query.Start(t.Context())
	require.NoError(t, err)
	assert.Equal(t, uint64(42), kept.GetPosition())
}
