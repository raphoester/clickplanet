package cpredis_test

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type ignored struct{}

func (ignored) Handle(context.Context, *wrapperspb.StringValue) error { return nil }

func groupOf(t *testing.T, config cpredis.Config, name string) cpredis.Group {
	t.Helper()

	group, ok := cpredis.GroupOf(subscribed[*wrapperspb.StringValue](t, connected(t, config, prometheus.NewRegistry()), name, ignored{}))
	require.True(t, ok)
	return group
}

func readAll(t *testing.T, group cpredis.Group, n int) []string {
	t.Helper()

	var positions []string
	for len(positions) < n {
		read, err := group.Read(t.Context())
		require.NoError(t, err)
		require.NotEmpty(t, read)
		positions = append(positions, read...)
	}
	return positions
}

func TestATrimKeepsWhatTheSlowestModuleHasNotHandled(t *testing.T) {
	server.Flush(t)
	config := server.Config()
	fast, slow := groupOf(t, config, "player-fast"), groupOf(t, config, "scoring-slow")
	events := make([]proto.Message, 1000)
	for i := range events {
		events[i] = wrapperspb.String("taken")
	}
	connected(t, config, prometheus.NewRegistry()).Publish(events...)

	for _, position := range readAll(t, fast, 1000) {
		fast.Ack(t.Context(), position)
	}
	slowly := readAll(t, slow, 1000)
	for _, position := range slowly[:600] {
		slow.Ack(t.Context(), position)
	}

	fast.Trim(t.Context())
	kept, err := fast.Len(t.Context())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, kept, int64(400), "what the slow module has not acked stays")
	assert.Less(t, kept, int64(1000))

	for _, position := range slowly[600:] {
		slow.Ack(t.Context(), position)
	}
	fast.Trim(t.Context())
	kept, err = fast.Len(t.Context())
	require.NoError(t, err)
	assert.Less(t, kept, int64(200), "a node of entries at most stays once every module acked")
}
