//go:build testing

package cps3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testImage  = "ghcr.io/versity/versitygw:v1.8.0"
	testPort   = "7070"
	testHealth = "/health"
	testKeyID  = "the-key-id"
	testSecret = "the-secret-key"
)

type TestServer struct {
	config Config
}

func StartTestServer(t testing.TB, bucket string) *TestServer {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testImage,
			ExposedPorts: []string{testPort + "/tcp"},
			// A directory under the root is a bucket: no client is needed to make one.
			Entrypoint: []string{"sh", "-c", "mkdir -p /data/" + bucket + " && exec versitygw --access " + testKeyID +
				" --secret " + testSecret + " --port :" + testPort + " --health " + testHealth + " posix /data"},
			// The port opens before the gateway answers: a request sent then got EOF in CI.
			WaitingFor: wait.ForHTTP(testHealth).WithPort(testPort + "/tcp").WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start the test bucket")

	endpoint, err := container.PortEndpoint(ctx, testPort, "http")
	require.NoError(t, err, "failed to get the test bucket endpoint")

	return &TestServer{config: Config{
		Endpoint: endpoint, Region: "us-east-1", Name: bucket, AccessKeyID: testKeyID, SecretAccessKey: testSecret,
	}}
}

func (s *TestServer) Config() Config {
	return s.config
}
