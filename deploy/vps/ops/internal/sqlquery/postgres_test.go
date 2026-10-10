package sqlquery_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raphoester/clickplanet.lol-ops/internal/sqlquery"
)

const (
	ownerPassword  = "owner-password"
	readerPassword = "reader-password"
)

type server struct {
	container testcontainers.Container
	endpoint  string
}

func (s server) applyRoleFile(t *testing.T, password string) {
	t.Helper()

	code, output, err := s.container.Exec(t.Context(), []string{
		"psql", "-U", "clickplanet", "-d", "clickplanet",
		"-v", "ON_ERROR_STOP=1", "-v", "password=" + password, "-f", "/ops/role.sql",
	})
	require.NoError(t, err)
	said, _ := io.ReadAll(output)
	require.Zero(t, code, "role.sql failed: %s", said)
}

func (s server) url(user, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s/clickplanet?sslmode=disable", user, password, s.endpoint)
}

func (s server) reader(t *testing.T, timeout time.Duration) *sqlquery.Postgres {
	t.Helper()

	reader, err := sqlquery.OpenPostgres(s.url("ops_reader", readerPassword), timeout)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	return reader
}

func (s server) open(t *testing.T, user, password string) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", s.url(user, password))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// The role is made by the same file the box runs, so the test covers what production grants.
func startServer(t *testing.T) server {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	roleFile, err := filepath.Abs(filepath.Join("..", "..", "postgres", "role.sql"))
	require.NoError(t, err)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "clickplanet",
				"POSTGRES_DB":       "clickplanet",
				"POSTGRES_PASSWORD": ownerPassword,
			},
			Files: []testcontainers.ContainerFile{
				{HostFilePath: roleFile, ContainerFilePath: "/ops/role.sql", FileMode: 0o644},
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start the test postgres")

	endpoint, err := container.PortEndpoint(ctx, "5432", "")
	require.NoError(t, err)
	started := server{container: container, endpoint: endpoint}
	started.applyRoleFile(t, readerPassword)

	_, err = started.open(t, "clickplanet", ownerPassword).ExecContext(ctx, `
		CREATE SCHEMA planet;
		CREATE TABLE planet.tiles (id integer PRIMARY KEY, country text NOT NULL);
		INSERT INTO planet.tiles VALUES (1, 'fr'), (2, 'bg'), (3, 'pt');
	`)
	require.NoError(t, err)

	return started
}

func TestAReaderOnlyReads(t *testing.T) {
	started := startServer(t)
	reader := started.reader(t, 10*time.Second)
	roomy := func(statement string) sqlquery.Query {
		return sqlquery.Query{Statement: statement, MaxRows: 100, MaxBytes: 1 << 20}
	}

	t.Run("a table in any schema is read", func(t *testing.T) {
		result, err := reader.Execute(t.Context(), roomy(`SELECT id, country FROM planet.tiles ORDER BY id`))

		require.NoError(t, err)
		assert.Equal(t, sqlquery.Result{
			Columns: []string{"id", "country"},
			Rows:    [][]any{{int64(1), "fr"}, {int64(2), "bg"}, {int64(3), "pt"}},
		}, result)
	})

	t.Run("a table made after the role is read too", func(t *testing.T) {
		_, err := started.open(t, "clickplanet", ownerPassword).ExecContext(t.Context(), `
			CREATE SCHEMA chat;
			CREATE TABLE chat.messages (body text);
			INSERT INTO chat.messages VALUES ('hello');
		`)
		require.NoError(t, err)

		result, err := reader.Execute(t.Context(), roomy(`SELECT body FROM chat.messages`))

		require.NoError(t, err)
		assert.Equal(t, [][]any{{"hello"}}, result.Rows)
	})

	t.Run("a statement that writes is refused", func(t *testing.T) {
		for _, statement := range []string{
			`INSERT INTO planet.tiles VALUES (4, 'de')`,
			`UPDATE planet.tiles SET country = 'de'`,
			`DELETE FROM planet.tiles`,
			`TRUNCATE planet.tiles`,
			`DROP TABLE planet.tiles`,
			`CREATE TABLE planet.mine (id integer)`,
			`WITH gone AS (DELETE FROM planet.tiles RETURNING id) SELECT * FROM gone`,
		} {
			_, err := reader.Execute(t.Context(), roomy(statement))

			require.ErrorIs(t, err, sqlquery.ErrRefused, statement)
		}

		result, err := reader.Execute(t.Context(), roomy(`SELECT count(*) FROM planet.tiles`))
		require.NoError(t, err)
		assert.Equal(t, [][]any{{int64(3)}}, result.Rows)
	})

	t.Run("several statements in one string are refused", func(t *testing.T) {
		_, err := reader.Execute(t.Context(), roomy(`SELECT 1; SELECT 2`))

		require.ErrorIs(t, err, sqlquery.ErrRefused)
	})

	t.Run("the role cannot write even once it turns read-only off", func(t *testing.T) {
		connection, err := started.open(t, "ops_reader", readerPassword).Conn(t.Context())
		require.NoError(t, err)
		defer func() { _ = connection.Close() }()
		_, err = connection.ExecContext(t.Context(), `SET default_transaction_read_only = off`)
		require.NoError(t, err)

		_, err = connection.ExecContext(t.Context(), `DELETE FROM planet.tiles`)

		require.ErrorContains(t, err, "permission denied")
	})

	t.Run("the role cannot run a program or read a file of the server", func(t *testing.T) {
		for _, statement := range []string{
			`COPY (SELECT 1) TO PROGRAM 'true'`,
			`SELECT pg_read_file('/etc/passwd')`,
		} {
			_, err := reader.Execute(t.Context(), roomy(statement))

			require.ErrorIs(t, err, sqlquery.ErrRefused, statement)
			assert.ErrorContains(t, err, "permission denied", statement)
		}
	})

	t.Run("more rows than the limit are cut and said so", func(t *testing.T) {
		result, err := reader.Execute(t.Context(), sqlquery.Query{
			Statement: `SELECT generate_series(1, 10)`, MaxRows: 3, MaxBytes: 1 << 20,
		})

		require.NoError(t, err)
		assert.True(t, result.Truncated)
		assert.Len(t, result.Rows, 3)
	})

	t.Run("more bytes than the limit are cut and said so", func(t *testing.T) {
		result, err := reader.Execute(t.Context(), sqlquery.Query{
			Statement: `SELECT repeat('x', 1000) FROM generate_series(1, 10)`, MaxRows: 100, MaxBytes: 2500,
		})

		require.NoError(t, err)
		assert.True(t, result.Truncated)
		assert.Len(t, result.Rows, 3)
	})

	t.Run("no row is an empty list, not a missing one", func(t *testing.T) {
		result, err := reader.Execute(t.Context(), roomy(`SELECT id FROM planet.tiles WHERE id < 0`))

		require.NoError(t, err)
		assert.Equal(t, sqlquery.Result{Columns: []string{"id"}, Rows: [][]any{}}, result)
	})

	t.Run("each type comes back as something json can carry", func(t *testing.T) {
		result, err := reader.Execute(t.Context(), roomy(`
			SELECT '\xdeadbeef'::bytea, '203.0.113.7'::inet, 1.50::numeric, NULL::text,
			       'NaN'::float8, '0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10'::uuid, '{"a":1}'::jsonb, true
		`))

		require.NoError(t, err)
		assert.Equal(t, [][]any{{
			`\xdeadbeef`, "203.0.113.7", "1.50", nil,
			"NaN", "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", `{"a": 1}`, true,
		}}, result.Rows)
	})

	t.Run("a statement that is not sql says what postgres said", func(t *testing.T) {
		_, err := reader.Execute(t.Context(), roomy(`SELEC 1`))

		require.ErrorIs(t, err, sqlquery.ErrRefused)
		assert.ErrorContains(t, err, "syntax error")
	})

	t.Run("a statement that outlasts its time is stopped", func(t *testing.T) {
		hasty := started.reader(t, 300*time.Millisecond)
		start := time.Now()

		_, err := hasty.Execute(t.Context(), roomy(`SELECT pg_sleep(30)`))

		require.Error(t, err)
		assert.Less(t, time.Since(start), 5*time.Second)
	})

	t.Run("the role file can run again, and then sets the new password", func(t *testing.T) {
		started.applyRoleFile(t, "rotated-password")
		defer started.applyRoleFile(t, readerPassword)

		rotated, err := sqlquery.OpenPostgres(started.url("ops_reader", "rotated-password"), 10*time.Second)
		require.NoError(t, err)
		defer func() { _ = rotated.Close() }()
		result, err := rotated.Execute(t.Context(), roomy(`SELECT count(*) FROM planet.tiles`))

		require.NoError(t, err)
		assert.Equal(t, [][]any{{int64(3)}}, result.Rows)
		_, err = reader.Execute(t.Context(), roomy(`SELECT 1`))
		require.Error(t, err)
	})

	t.Run("a lock taken by one statement is gone before the next", func(t *testing.T) {
		_, err := reader.Execute(t.Context(), roomy(`SELECT pg_advisory_lock(42)`))
		require.NoError(t, err)

		result, err := reader.Execute(t.Context(), roomy(
			`SELECT count(*) FROM pg_locks WHERE locktype = 'advisory'`,
		))

		require.NoError(t, err)
		assert.Equal(t, [][]any{{int64(0)}}, result.Rows)
	})
}
