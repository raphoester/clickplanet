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

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testImage = "postgres:16-alpine"

// One container per test binary, migrated once; the testcontainers reaper removes it on exit.
var shared struct {
	once   sync.Once
	client *Postgres
	err    error
}

// ForTests hands back a migrated database with every table emptied. Call it from SetupTest; it needs Docker.
func ForTests(t testing.TB, migrations fs.FS) *Postgres {
	t.Helper()

	shared.once.Do(func() {
		shared.client, shared.err = startContainer(migrations)
	})
	if shared.err != nil {
		t.Fatalf("failed to start the test postgres: %v", shared.err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	if err := shared.client.purge(ctx); err != nil {
		t.Fatalf("failed to purge the test postgres: %v", err)
	}

	return shared.client
}

func startContainer(migrations fs.FS) (*Postgres, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testImage,
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_PASSWORD": "postgres",
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

	endpoint, err := container.PortEndpoint(ctx, "5432", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get container endpoint: %w", err)
	}

	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to split host and port: %w", err)
	}

	client := New(Config{
		Host:     host,
		Port:     port,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
	})

	if err := client.Migrate(migrations); err != nil {
		return nil, err
	}

	if err := client.ConnectCtx(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

// purge empties every table but migrate's own, so each test starts from the schema and nothing else.
func (p *Postgres) purge(ctx context.Context) error {
	rows, err := p.QueryContext(ctx, `
		SELECT quote_ident(table_name)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		  AND table_name <> 'schema_migrations'
	`)
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
