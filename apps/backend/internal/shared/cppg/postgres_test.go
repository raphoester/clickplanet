package cppg_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestThePasswordStaysOutOfAPrintedConfig(t *testing.T) {
	config := struct{ Database cppg.Config }{cppg.Config{Host: "postgres", Password: "hunter2"}}

	printed := fmt.Sprintf("%+v", config)

	assert.NotContains(t, printed, "hunter2")
	assert.Contains(t, printed, "postgres")
}

func TestAnIncompleteConfigNamesEveryMissingKey(t *testing.T) {
	err := cppg.Config{Host: "postgres", Port: "5432"}.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "[user dbName sslMode schema] is empty")
}

func TestASchemaThatIsNotAPlainIdentifierIsRefused(t *testing.T) {
	config := cppg.Config{Host: "h", Port: "5432", User: "u", DBName: "d", SSLMode: "disable", Schema: "planet; drop"}

	require.ErrorContains(t, config.Validate(), "not a lowercase identifier")
}

func TestACompleteConfigIsValid(t *testing.T) {
	require.NoError(t, cppg.Config{Host: "h", Port: "5432", User: "u", DBName: "d", SSLMode: "disable", Schema: "planet"}.Validate())
}

func TestEachSchemaHoldsItsOwnTablesAndMigrationHistory(t *testing.T) {
	migrations := fstest.MapFS{
		"1_create_things.up.sql":   {Data: []byte(`CREATE TABLE things (id integer PRIMARY KEY)`)},
		"1_create_things.down.sql": {Data: []byte(`DROP TABLE things`)},
	}
	ctx := t.Context()

	database := server.CreateDatabase(t)
	first := database.OpenSchema(t, "first", migrations)
	second := database.OpenSchema(t, "second", migrations)

	_, err := first.On(ctx).ExecContext(ctx, `INSERT INTO things VALUES (1)`)
	require.NoError(t, err)

	var count int
	require.NoError(t, second.On(ctx).QueryRowContext(ctx, `SELECT count(*) FROM things`).Scan(&count))
	assert.Zero(t, count, "the same unqualified table name is a different table in each schema")

	var inPublic *string
	require.NoError(t, first.On(ctx).QueryRowContext(ctx, `SELECT to_regclass('public.things')::text`).Scan(&inPublic))
	assert.Nil(t, inPublic, "nothing lands in public")

	var version int
	require.NoError(t, second.On(ctx).QueryRowContext(ctx, `SELECT version FROM second.schema_migrations`).Scan(&version))
	assert.Equal(t, 1, version)
}

type countingRunner struct {
	name    string
	stopped *atomic.Int32
}

func (r countingRunner) Name() string { return r.name }

func (r countingRunner) Run(ctx context.Context) {
	<-ctx.Done()
	r.stopped.Add(1)
}

func TestCloseAfterClosesThePoolOnceEveryRunnerStopped(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "closing", fstest.MapFS{
		"1_create_things.up.sql":   {Data: []byte(`CREATE TABLE things (id integer PRIMARY KEY)`)},
		"1_create_things.down.sql": {Data: []byte(`DROP TABLE things`)},
	})
	stopped := &atomic.Int32{}
	runner := cppg.CloseAfter(db, slog.New(slog.DiscardHandler),
		countingRunner{name: "tiles", stopped: stopped}, countingRunner{name: "ledger", stopped: stopped})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	runner.Run(ctx)

	assert.Equal(t, int32(2), stopped.Load())
	assert.Equal(t, "tiles+ledger", runner.Name())
	_, err := db.On(t.Context()).ExecContext(t.Context(), `SELECT 1`)
	require.ErrorContains(t, err, "database is closed")
}

func TestAPoolKeepsEveryConnectionItOpened(t *testing.T) {
	config := server.CreateDatabase(t).ConfigFor("pooled")
	most := 4
	config.Pool.MaxOpenConns = &most
	db := cppg.New(config)
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })

	var busy, done sync.WaitGroup
	release := make(chan struct{})
	for range most {
		busy.Add(1)
		done.Go(func() {
			assert.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
				busy.Done()
				<-release
				return ctx.Err()
			}))
		})
	}
	busy.Wait()
	close(release)
	done.Wait()

	stats := db.Stats()
	assert.Equal(t, most, stats.Idle)
	assert.Zero(t, stats.MaxIdleClosed, "no connection closed for want of an idle slot")
}
