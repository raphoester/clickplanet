//go:build testing

package cppg

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testImage = "postgres:16-alpine"

// One container per test binary; the testcontainers reaper removes it on exit.
var shared struct {
	once    sync.Once
	config  Config
	err     error
	mu      sync.Mutex
	schemas map[string]*Postgres
}

// ForTests hands back a client inside schema, migrated once per binary, with every table emptied. Call it from SetupTest; it needs Docker.
func ForTests(t testing.TB, schema string, migrations fs.FS) *Postgres {
	t.Helper()

	shared.once.Do(startShared)
	require.NoError(t, shared.err, "failed to start the test postgres")

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	shared.mu.Lock()
	defer shared.mu.Unlock()

	client, ok := shared.schemas[schema]
	if !ok {
		var err error
		client, err = openSchema(ctx, shared.config, schema, migrations)
		require.NoError(t, err, "failed to open the test schema")
		shared.schemas[schema] = client
	}

	require.NoError(t, client.purge(ctx), "failed to purge the test postgres")

	return client
}

// startShared starts the container and records how to reach it, once per binary.
func startShared() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	shared.schemas = map[string]*Postgres{}

	container, err := startContainer(ctx)
	if err != nil {
		shared.err = err
		return
	}

	endpoint, err := container.PortEndpoint(ctx, "5432", "")
	if err != nil {
		shared.err = fmt.Errorf("failed to get container endpoint: %w", err)
		return
	}

	shared.config, shared.err = configFor(endpoint)
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

// purge empties every table in the client's schema but migrate's own.
func (p *Postgres) purge(ctx context.Context) error {
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
