//go:build testing

package cppg

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testImage = "postgres:16-alpine"

// TestServer is a postgres in a container, for the tests of one suite.
type TestServer struct {
	config Config
}

// StartTestServer starts a container and stops it when t ends. Call it from SetupSuite; it needs Docker.
func StartTestServer(t testing.TB) *TestServer {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	container, err := startContainer(ctx)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start the test postgres")

	endpoint, err := container.PortEndpoint(ctx, "5432", "")
	require.NoError(t, err, "failed to get the test postgres endpoint")

	config, err := configFor(endpoint)
	require.NoError(t, err, "failed to read the test postgres endpoint")

	return &TestServer{config: config}
}

// OpenSchema connects inside schema and migrates it. The client closes when t ends.
func (s *TestServer) OpenSchema(t testing.TB, schema string, migrations fs.FS) *Postgres {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	client, err := openSchema(ctx, s.config, schema, migrations)
	require.NoError(t, err, "failed to open the test schema")
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func startContainer(ctx context.Context) (testcontainers.Container, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testImage,
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_PASSWORD": testPassword,
			},
			// The image starts postgres twice: once to run its init scripts, then for real.
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	return container, nil
}

const testPassword = "postgres"

// configFor is the connection to the container listening on endpoint, a host:port.
func configFor(endpoint string) (Config, error) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return Config{}, fmt.Errorf("failed to split host and port: %w", err)
	}

	return Config{
		Host:     host,
		Port:     port,
		User:     "postgres",
		Password: testPassword,
		DBName:   "postgres",
		SSLMode:  "disable",
	}, nil
}

// openSchema connects inside schema and migrates it.
func openSchema(ctx context.Context, config Config, schema string, migrations fs.FS) (*Postgres, error) {
	config.Schema = schema
	client := New(config)

	if err := client.ConnectCtx(ctx); err != nil {
		return nil, err
	}

	if err := client.Migrate(ctx, migrations); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

// Purge empties every table in the client's schema but migrate's own. Call it from SetupTest.
func (p *Postgres) Purge(ctx context.Context) error {
	rows, err := p.QueryContext(ctx, `
		SELECT quote_ident(table_schema) || '.' || quote_ident(table_name)
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		  AND table_name <> 'schema_migrations'
	`, p.config.Schema)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	if len(tables) == 0 {
		return nil
	}

	_, err = p.ExecContext(ctx, "TRUNCATE TABLE "+strings.Join(tables, ", ")+" CASCADE")

	return err
}
