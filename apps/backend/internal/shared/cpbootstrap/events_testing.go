//go:build testing

package cpbootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/proto"
)

type RecordedEvents struct {
	mu     sync.Mutex
	events []proto.Message
}

func NewRecordedEvents() *RecordedEvents {
	return &RecordedEvents{}
}

func (r *RecordedEvents) Publish(events ...proto.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, events...)
}

func (r *RecordedEvents) Published() []proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]proto.Message(nil), r.events...)
}

const testRedisImage = "redis:7.4-alpine"

func StartTestRedis(t testing.TB) EventsConfig {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testRedisImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForAll(wait.ForLog("Ready to accept connections"), wait.ForListeningPort("6379/tcp")).
				WithDeadline(time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start the test redis")

	endpoint, err := container.PortEndpoint(ctx, "6379", "")
	require.NoError(t, err, "failed to get the test redis endpoint")

	return EventsConfig{Address: endpoint, RetryAfter: 50 * time.Millisecond}
}
