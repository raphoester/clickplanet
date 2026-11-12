//go:build testing

// Package cps3test is apart from cps3 so that only the test binaries that start a container pay testcontainers' init.
package cps3test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
)

const (
	image  = "ghcr.io/versity/versitygw:v1.8.0"
	port   = "7070"
	health = "/health"
	keyID  = "the-key-id"
	secret = "the-secret-key"
)

type Server struct {
	config cps3.Config
}

func StartServer(t testing.TB, bucket string) *Server {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{port + "/tcp"},
			// A directory under the root is a bucket: no client is needed to make one.
			Entrypoint: []string{"sh", "-c", "mkdir -p /data/" + bucket + " && exec versitygw --access " + keyID +
				" --secret " + secret + " --port :" + port + " --health " + health + " posix /data"},
			// The port opens before the gateway answers: a request sent then got EOF in CI.
			WaitingFor: wait.ForHTTP(health).WithPort(port + "/tcp").WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start the test bucket")

	endpoint, err := container.PortEndpoint(ctx, port, "http")
	require.NoError(t, err, "failed to get the test bucket endpoint")

	return &Server{config: cps3.Config{
		Endpoint: endpoint, Region: "us-east-1", Name: bucket, AccessKeyID: keyID, SecretAccessKey: secret,
	}}
}

func (s *Server) Config() cps3.Config {
	return s.config
}
