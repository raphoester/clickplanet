package migrations_test

import (
	"database/sql"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

const mailboxMigration = "20261115123255_identity_mailbox"

var linked = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// The schema as production holds it before the mailbox: every migration older than it.
func beforeTheMailbox(t *testing.T) fs.FS {
	t.Helper()

	older := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() >= mailboxMigration {
			continue
		}
		data, err := fs.ReadFile(migrations.FS, entry.Name())
		require.NoError(t, err)
		older[entry.Name()] = &fstest.MapFile{Data: data}
	}
	return older
}

type kept struct {
	db *cppg.Postgres
}

func keptBeforeTheMailbox(t *testing.T) kept {
	t.Helper()
	return kept{db: cppgtest.StartServer(t).OpenSchema(t, "auth", beforeTheMailbox(t))}
}

func (k kept) identity(t *testing.T, account byte, provider string, subject string, email string, at time.Time) {
	t.Helper()

	ctx := t.Context()
	id := uuid.UUID(accounts.AccountID{15: account})
	_, err := k.db.On(ctx).ExecContext(ctx, `
		INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2) ON CONFLICT (id) DO NOTHING
	`, id, at)
	require.NoError(t, err)
	_, err = k.db.On(ctx).ExecContext(ctx, `
		INSERT INTO identities (provider, subject, account_id, email, email_verified, linked_at)
		VALUES ($1::identity_provider, $2, $3, NULLIF($4, ''), $4 <> '', $5)
	`, provider, subject, id, email, at)
	require.NoError(t, err)
}

func (k kept) code(t *testing.T, account byte, address string, at time.Time) {
	t.Helper()
	k.identity(t, account, "email", address, address, at)
}

func (k kept) migrated(t *testing.T) {
	t.Helper()
	require.NoError(t, k.db.Migrate(t.Context(), migrations.FS))
}

func (k kept) subjectOf(t *testing.T, account byte) string {
	t.Helper()

	var subject string
	require.NoError(t, k.db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT subject FROM identities WHERE provider = 'email' AND account_id = $1`,
		uuid.UUID(accounts.AccountID{15: account})).Scan(&subject))
	return subject
}

func TestTheMigrationFoldsEachAddressAsMailboxOfDoes(t *testing.T) {
	k := keptBeforeTheMailbox(t)
	addresses := []string{
		"John.Doe+x@Gmail.com", "j.o.h.n@googlemail.com", "+tag@gmail.com", "...@gmail.com", "a.b+c+d@gmail.com",
		"jane.doe+1@outlook.com", "jane+1@outlook.fr", "jane+1@outlook.de", "jane+1@outlook.es", "jane+1@outlook.it",
		"jane.doe+1@hotmail.com", "jane+1@hotmail.fr", "jane+1@hotmail.co.uk", "jane+1@hotmail.de", "jane+1@hotmail.it",
		"jane+1@hotmail.es", "jane+1@live.com", "jane+1@live.fr", "jane+1@live.co.uk", "jane+1@msn.com",
		"jane+1@icloud.com", "jane+1@me.com", "jane+1@mac.com",
		"jane+1@proton.me", "jane+1@protonmail.com", "jane+1@protonmail.ch", "jane+1@pm.me",
		"jane+1@fastmail.com", "jane+1@fastmail.fm", "ivan+1@yandex.ru", "ivan+1@yandex.com", "ivan+1@ya.ru",
		"+@outlook.com", "jane+1@yahoo.com", "Jane.Doe+1@Example.com", "jane+1@mail.gmail.com", "nobody",
	}
	for i, address := range addresses {
		k.identity(t, byte(i+1), "discord", "discord-"+address, address, linked)
	}
	k.identity(t, 200, "discord", "unverified", "", linked)

	k.migrated(t)

	for i, address := range addresses {
		var mailbox string
		require.NoError(t, k.db.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT mailbox FROM identities WHERE account_id = $1`, uuid.UUID(accounts.AccountID{15: byte(i + 1)})).Scan(&mailbox))
		assert.Equal(t, accounts.MailboxOf(address).String(), mailbox, address)
	}
	var unverified sql.NullString
	require.NoError(t, k.db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT mailbox FROM identities WHERE subject = 'unverified'`).Scan(&unverified))
	assert.False(t, unverified.Valid, "an identity with no verified address has no mailbox")
}

func TestTheOwnerOfAMailboxKeepsItAndEveryOtherAccountsCodeGivesItUp(t *testing.T) {
	const (
		owner, sameName, dotted, firstProton, laterProton, alone, foreign = 1, 2, 3, 4, 5, 6, 7
	)
	k := keptBeforeTheMailbox(t)
	k.identity(t, owner, "google", "google-owner", "John.Doe@gmail.com", linked)
	k.code(t, sameName, "johndoe@gmail.com", linked.Add(time.Hour))
	k.code(t, dotted, "john.doe+x@gmail.com", linked.Add(2*time.Hour))
	k.code(t, firstProton, "ada+1@proton.me", linked)
	k.code(t, laterProton, "ada@proton.me", linked.Add(time.Hour))
	k.code(t, alone, "solo+x@outlook.com", linked)
	k.code(t, foreign, "x+y@example.com", linked)

	k.migrated(t)

	ctx := t.Context()
	store := postgres_account_store.New(k.db)
	found, err := store.AccountOfMailbox(ctx, accounts.MailboxOf("j.o.h.n.d.o.e+new@gmail.com"))
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: owner}, found.ID(), "the oldest identity's account owns the mailbox")
	_, err = store.Identity(ctx, "email", "johndoe@gmail.com")
	require.ErrorIs(t, err, accounts.ErrIdentityNotFound,
		"no code finds an account beside the owner's: the next one joins the owner, who has no code yet")
	assert.Equal(t, "alias:"+accounts.AccountID{15: sameName}.String()+":johndoe@gmail.com", k.subjectOf(t, sameName))
	assert.Equal(t, "alias:"+accounts.AccountID{15: dotted}.String()+":john.doe+x@gmail.com", k.subjectOf(t, dotted))

	identity, err := store.Identity(ctx, "email", "ada@proton.me")
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: firstProton}, identity.Account(), "the oldest code keeps the mailbox")
	assert.Equal(t, "ada+1@proton.me", identity.Email(), "and its address")
	assert.Equal(t, "alias:"+accounts.AccountID{15: laterProton}.String()+":ada@proton.me", k.subjectOf(t, laterProton))

	identity, err = store.Identity(ctx, "email", "solo@outlook.com")
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: alone}, identity.Account())
	assert.Equal(t, "x+y@example.com", k.subjectOf(t, foreign), "an unknown domain's address is its own mailbox")

	var emails int
	require.NoError(t, k.db.On(ctx).QueryRowContext(ctx,
		`SELECT count(*) FROM identities WHERE email IN ('johndoe@gmail.com', 'john.doe+x@gmail.com', 'ada@proton.me')`).Scan(&emails))
	assert.Equal(t, 3, emails, "every address given up is still kept, for the operator's search")
}

func TestTheDownMigrationGivesEachCodeItsAddressBack(t *testing.T) {
	k := keptBeforeTheMailbox(t)
	k.code(t, 1, "ada+1@proton.me", linked)
	k.code(t, 2, "ada@proton.me", linked.Add(time.Hour))
	k.migrated(t)

	down, err := fs.ReadFile(migrations.FS, mailboxMigration+".down.sql")
	require.NoError(t, err)
	_, err = k.db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.Equal(t, "ada+1@proton.me", k.subjectOf(t, 1))
	assert.Equal(t, "ada@proton.me", k.subjectOf(t, 2))
}
