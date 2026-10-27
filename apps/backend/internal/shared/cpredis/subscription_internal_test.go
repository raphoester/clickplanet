package cpredis

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ignored struct{}

func (ignored) Handle(context.Context, *wrapperspb.StringValue) error { return nil }

func groupOf(t *testing.T, config Config, name string) *group {
	t.Helper()

	streams := New(config, prometheus.NewRegistry())
	require.NoError(t, streams.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = streams.Close() })

	runner, err := Subscribe[*wrapperspb.StringValue](t.Context(), streams, name, ignored{})
	require.NoError(t, err)
	subscribed, ok := runner.(*subscription[*wrapperspb.StringValue])
	require.True(t, ok)
	return subscribed.group
}

func readAll(t *testing.T, subscribed *group, n int) []string {
	t.Helper()

	var positions []string
	for len(positions) < n {
		messages, err := subscribed.read(t.Context(), noBlock)
		require.NoError(t, err)
		require.NotEmpty(t, messages)
		for _, message := range messages {
			positions = append(positions, message.ID)
		}
	}
	return positions
}

func TestATrimKeepsWhatTheSlowestModuleHasNotHandled(t *testing.T) {
	config := StartTestServer(t).Config()
	fast, slow := groupOf(t, config, "player-fast"), groupOf(t, config, "scoring-slow")
	publisher := New(config, prometheus.NewRegistry())
	require.NoError(t, publisher.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = publisher.Close() })
	events := make([]proto.Message, 1000)
	for i := range events {
		events[i] = wrapperspb.String("taken")
	}
	publisher.Publish(events...)

	for _, position := range readAll(t, fast, 1000) {
		fast.ack(t.Context(), position)
	}
	slowly := readAll(t, slow, 1000)
	for _, position := range slowly[:600] {
		slow.ack(t.Context(), position)
	}

	fast.trim(t.Context())
	kept, err := fast.client.XLen(t.Context(), fast.stream).Result()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, kept, int64(400), "what the slow module has not acked stays")
	assert.Less(t, kept, int64(1000))

	for _, position := range slowly[600:] {
		slow.ack(t.Context(), position)
	}
	fast.trim(t.Context())
	kept, err = fast.client.XLen(t.Context(), fast.stream).Result()
	require.NoError(t, err)
	assert.Less(t, kept, int64(200), "a node of entries at most stays once every module acked")
}
