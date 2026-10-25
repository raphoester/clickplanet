package cpoutbox_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var notesMigrations = fstest.MapFS{
	"1_notes.up.sql": &fstest.MapFile{Data: []byte(`
		CREATE TYPE outbox_event AS ENUM ('google.protobuf.StringValue');
		CREATE TABLE outbox (position bigserial PRIMARY KEY, event outbox_event NOT NULL, payload bytea NOT NULL);
		CREATE TABLE notes (name text PRIMARY KEY);
	`)},
	"1_notes.down.sql": &fstest.MapFile{Data: []byte(`DROP TABLE notes; DROP TABLE outbox; DROP TYPE outbox_event;`)},
}

type notes struct {
	db cppg.Conn
}

func (n notes) Save(ctx context.Context, names ...string) error {
	return n.db.InTx(ctx, func(ctx context.Context) error {
		for _, name := range names {
			if _, err := n.db.On(ctx).ExecContext(ctx, `INSERT INTO notes (name) VALUES ($1)`, name); err != nil {
				return err //nolint:wrapcheck // a test store.
			}
		}
		return nil
	})
}

func open(t *testing.T, server *cppg.TestServer, schema string) (*cppg.Postgres, notes, cpatomic.Outbox) {
	t.Helper()

	db := server.OpenSchema(t, schema, notesMigrations)
	return db, notes{db: db}, cpoutbox.New(db)
}

func count(t *testing.T, db cppg.Conn, table string) int {
	t.Helper()

	var rows int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT count(*) FROM `+table).Scan(&rows))
	return rows
}

func TestAWriteAndItsEventCommitTogether(t *testing.T) {
	db, store, outbox := open(t, cppg.StartTestServer(t), "together")

	require.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
		if err := store.Save(ctx, "a", "b"); err != nil {
			return err
		}
		return outbox.Record(ctx, wrapperspb.String("saved"))
	}))

	assert.Equal(t, 2, count(t, db, "notes"))
	assert.Equal(t, 1, count(t, db, "outbox"))
}

func TestAFailedTransactionKeepsNeitherTheWriteNorTheEvent(t *testing.T) {
	db, store, outbox := open(t, cppg.StartTestServer(t), "failed")

	require.Error(t, db.InTx(t.Context(), func(ctx context.Context) error {
		if err := outbox.Record(ctx, wrapperspb.String("saved")); err != nil {
			return err //nolint:wrapcheck // a test.
		}
		if err := store.Save(ctx, "a"); err != nil {
			return err
		}
		return store.Save(ctx, "a")
	}))

	assert.Zero(t, count(t, db, "notes"), "the store's own InTx joined the open one")
	assert.Zero(t, count(t, db, "outbox"))
}

func TestAPanicKeepsNothing(t *testing.T) {
	db, store, outbox := open(t, cppg.StartTestServer(t), "panicked")

	assert.Panics(t, func() {
		_ = db.InTx(t.Context(), func(ctx context.Context) error {
			_ = store.Save(ctx, "a")
			_ = outbox.Record(ctx, wrapperspb.String("saved"))
			panic("a bug")
		})
	})

	assert.Zero(t, count(t, db, "notes"))
	assert.Zero(t, count(t, db, "outbox"))
}

func TestAnEventIsNeverRecordedOutsideATransaction(t *testing.T) {
	db, _, outbox := open(t, cppg.StartTestServer(t), "outside")

	require.ErrorIs(t, outbox.Record(t.Context(), wrapperspb.String("alone")), cppg.ErrNoTransaction)
	assert.Zero(t, count(t, db, "outbox"))
}

func TestATransactionOfOneDatabaseNeverReachesAnother(t *testing.T) {
	server := cppg.StartTestServer(t)
	auth, _, _ := open(t, server, "auth")
	seasons, seasonNotes, seasonsOutbox := open(t, server, "seasons")

	require.Error(t, auth.InTx(t.Context(), func(ctx context.Context) error {
		if err := seasonNotes.Save(ctx, "a"); err != nil {
			return err
		}
		require.ErrorIs(t, seasonsOutbox.Record(ctx, wrapperspb.String("saved")), cppg.ErrNoTransaction)
		return context.Canceled
	}))

	assert.Equal(t, 1, count(t, seasons, "notes"), "seasons wrote in its own transaction")
}

func TestAWriteOnTheOuterContextIsNotInTheTransaction(t *testing.T) {
	db, store, outbox := open(t, cppg.StartTestServer(t), "outer")
	outer := t.Context()

	require.Error(t, db.InTx(outer, func(ctx context.Context) error {
		if err := store.Save(outer, "a"); err != nil {
			return err
		}
		if err := outbox.Record(ctx, wrapperspb.String("saved")); err != nil {
			return err //nolint:wrapcheck // a test.
		}
		return context.Canceled
	}))

	assert.Equal(t, 1, count(t, db, "notes"), "the known gap: the outer ctx carries no transaction")
	assert.Zero(t, count(t, db, "outbox"))
}

func TestTheFakeOutboxDropsWhatAFailedTransactionRecorded(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}
	transactioner := cpatomic.FakeTransactioner{}

	require.Error(t, transactioner.InTx(t.Context(), func(ctx context.Context) error {
		_ = outbox.Record(ctx, wrapperspb.String("lost"))
		return context.Canceled
	}))
	require.NoError(t, transactioner.InTx(t.Context(), func(ctx context.Context) error {
		return outbox.Record(ctx, wrapperspb.String("kept"))
	}))
	require.ErrorIs(t, outbox.Record(t.Context(), wrapperspb.String("alone")), cpatomic.ErrNoTransaction)

	require.Len(t, outbox.Recorded(), 1)
	assert.Equal(t, "kept", outbox.Recorded()[0].(*wrapperspb.StringValue).GetValue()) //nolint:forcetypeassert // strings only.
}
