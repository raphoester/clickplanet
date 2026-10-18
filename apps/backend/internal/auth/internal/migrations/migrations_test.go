package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const identityProvider = "20261012120000"

func TestEveryIdentityKeptBeforeTheTypeKeepsItsProvider(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "auth", cppg.MigrationsBefore(t, migrations.FS, identityProvider))
	at := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	account := uuid.UUID{15: 1}
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2)`, account, at)
	require.NoError(t, err)
	providers := []string{"google", "discord", "email"}
	insert := `INSERT INTO identities (provider, subject, account_id, email_verified, linked_at) VALUES ($1, $2, $3, false, $4)`
	for _, provider := range providers {
		_, err := db.ExecContext(t.Context(), insert, provider, "the subject of "+provider, account, at)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	for _, provider := range providers {
		var kept string
		require.NoError(t, db.QueryRowContext(t.Context(),
			`SELECT provider FROM identities WHERE subject = $1`, "the subject of "+provider).Scan(&kept))
		assert.Equal(t, provider, kept)
	}
	_, err = db.ExecContext(t.Context(), insert, "google", "the subject of google", account, at)
	require.Error(t, err, "one provider still names a subject once")
	_, err = db.ExecContext(t.Context(), insert, "github", "the subject of github", account, at)
	require.Error(t, err, "a provider is a value of the type before an identity has it")
}

const accountOperator = "20261017120000"

func TestNoAccountKeptBeforeTheFlagIsAnOperator(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "auth", cppg.MigrationsBefore(t, migrations.FS, accountOperator))
	at := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	account := uuid.UUID{15: 1}
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2)`, account, at)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var operator bool
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT operator FROM accounts WHERE id = $1`, account).Scan(&operator))
	assert.False(t, operator)
}
