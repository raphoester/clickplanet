//go:build testing

// Package cpredistest is apart from cpredis so that only the test binaries that start a container pay testcontainers' init.
package cpredistest

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const image = "redis:7.4-alpine"

type Server struct {
	container testcontainers.Container
	config    cpredis.Config
}

func StartServer(t testing.TB) *Server {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	server := &Server{}
	err := server.start(ctx)
	testcontainers.CleanupContainer(t, server.container)
	require.NoError(t, err, "failed to start the test redis")

	return server
}

func (s *Server) Run(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	err := s.start(ctx)
	cancel()
	if err != nil {
		_ = testcontainers.TerminateContainer(s.container)
		log.Fatalf("failed to start the test redis: %v", err)
	}

	m.Run()

	if err := testcontainers.TerminateContainer(s.container); err != nil {
		log.Printf("failed to stop the test redis: %v", err)
	}
}

func (s *Server) Config() cpredis.Config {
	return s.config
}

func (s *Server) Flush(t testing.TB) {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: s.config.Address})
	defer func() { _ = client.Close() }()
	require.NoError(t, client.FlushAll(t.Context()).Err(), "failed to empty the test redis")
}

func (s *Server) start(ctx context.Context) error {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForAll(wait.ForLog("Ready to accept connections"), wait.ForListeningPort("6379/tcp")).
				WithDeadline(time.Minute),
		},
		Started: true,
	})
	s.container = container
	if err != nil {
		return fmt.Errorf("failed to start redis container: %w", err)
	}

	endpoint, err := container.PortEndpoint(ctx, "6379", "")
	if err != nil {
		return fmt.Errorf("failed to get the test redis endpoint: %w", err)
	}

	s.config = cpredis.Config{Address: endpoint, RetryAfter: 50 * time.Millisecond}
	return nil
}
