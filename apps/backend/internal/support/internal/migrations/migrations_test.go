package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
)

const noteDiagnostics = "20261012100000"

func TestANoteKeptBeforeDiagnosticsIsReadWithEmptyOnes(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "support", cppg.MigrationsBefore(t, migrations.FS, noteDiagnostics))
	ada := notes.AccountID{15: 1}
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO notes (id, kind, account_id, text, submitted_at) VALUES ($1, 'suggestion', $2, 'a night mode', $3)`,
		uuid.UUID{15: 1}, uuid.UUID(ada), at)
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO notes (id, kind, text, submitted_at) VALUES ($1, 'suggestion', 'more flags', $2)`,
		uuid.UUID{15: 2}, at.Add(time.Minute))
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	sent, err := postgres_note_store.New(db).SentBy(t.Context(), ada)
	require.NoError(t, err)
	require.Len(t, sent, 1)
	assert.Equal(t, notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode", At: at,
	}, sent[0].TakeSnapshot())

	var empty int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT count(*) FROM notes WHERE diagnostics = '{}'`).Scan(&empty))
	assert.Equal(t, 2, empty, "the anonymized note too")
}
