//go:build testing

// Package cppgtest is apart from cppg so that only the test binaries that start a container pay testcontainers' init.
package cppgtest

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	image    = "postgres:16-alpine"
	password = "postgres"
)

type Database struct {
	config cppg.Config
}

func (d Database) ConfigFor(schema string) cppg.Config {
	config := d.config
	config.Schema = schema
	return config
}

func (d Database) OpenSchema(t testing.TB, schema string, migrations fs.FS) *cppg.Postgres {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	client := cppg.New(d.ConfigFor(schema))
	require.NoError(t, client.ConnectCtx(ctx), "failed to open the test schema")
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Migrate(ctx, migrations), "failed to migrate the test schema")

	return client
}

type Server struct {
	Database

	container testcontainers.Container
	databases atomic.Int64
}

func StartServer(t testing.TB) *Server {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	server := &Server{}
	err := server.start(ctx)
	testcontainers.CleanupContainer(t, server.container)
	require.NoError(t, err, "failed to start the test postgres")

	return server
}

func (s *Server) Run(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	err := s.start(ctx)
	cancel()
	if err != nil {
		_ = testcontainers.TerminateContainer(s.container)
		log.Fatalf("failed to start the test postgres: %v", err)
	}

	m.Run()

	if err := testcontainers.TerminateContainer(s.container); err != nil {
		log.Printf("failed to stop the test postgres: %v", err)
	}
}

func (s *Server) CreateDatabase(t testing.TB) Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	name := fmt.Sprintf("test_%d", s.databases.Add(1))
	admin := cppg.New(s.ConfigFor("public"))
	require.NoError(t, admin.ConnectCtx(ctx), "failed to reach the test postgres")
	defer func() { _ = admin.Close() }()
	_, err := admin.On(ctx).ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(name))
	require.NoError(t, err, "failed to create the test database")

	config := s.config
	config.DBName = name
	return Database{config: config}
}

func (s *Server) start(ctx context.Context) error {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{"5432/tcp"},
			Env:          map[string]string{"POSTGRES_PASSWORD": password},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	s.container = container
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	endpoint, err := container.PortEndpoint(ctx, "5432", "")
	if err != nil {
		return fmt.Errorf("failed to get the test postgres endpoint: %w", err)
	}

	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("failed to split host and port: %w", err)
	}

	s.config = cppg.Config{
		Host:     host,
		Port:     port,
		User:     "postgres",
		Password: password,
		DBName:   "postgres",
		SSLMode:  "disable",
	}
	return nil
}
