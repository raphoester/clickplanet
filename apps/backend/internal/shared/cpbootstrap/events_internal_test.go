package cpbootstrap

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func sealedBus(t *testing.T, subscribers ...string) (*eventBus, []*group) {
	t.Helper()

	bus, err := newEventBus(t.Context(), StartTestRedis(t), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = bus.close() })

	groups := make([]*group, 0, len(subscribers))
	for _, name := range subscribers {
		subscribed, err := bus.register("google.protobuf.StringValue", name)
		require.NoError(t, err)
		groups = append(groups, subscribed)
	}
	require.NoError(t, bus.seal(t.Context()))

	return bus, groups
}

func publishMany(bus *eventBus, n int) {
	events := make([]proto.Message, n)
	for i := range events {
		events[i] = wrapperspb.String("taken")
	}
	bus.Publish(events...)
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

func TestATrimKeepsWhatTheSlowestSubscriberHasNotHandled(t *testing.T) {
	bus, groups := sealedBus(t, "fast", "slow")
	stream := streamOf("google.protobuf.StringValue")
	publishMany(bus, 1000)

	for _, position := range readAll(t, groups[0], 1000) {
		groups[0].ack(t.Context(), position)
	}
	slow := readAll(t, groups[1], 1000)
	for _, position := range slow[:600] {
		groups[1].ack(t.Context(), position)
	}

	bus.trim(t.Context())
	kept, err := bus.client.XLen(t.Context(), stream).Result()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, kept, int64(400), "what slow has not acked stays")
	assert.Less(t, kept, int64(1000))

	for _, position := range slow[600:] {
		groups[1].ack(t.Context(), position)
	}
	bus.trim(t.Context())
	kept, err = bus.client.XLen(t.Context(), stream).Result()
	require.NoError(t, err)
	assert.Less(t, kept, int64(200), "a node of entries at most stays once everyone acked")
}

func TestAGroupNobodySubscribesWithIsDestroyedAtTheBoot(t *testing.T) {
	config := StartTestRedis(t)
	stream := streamOf("google.protobuf.StringValue")

	first, err := newEventBus(t.Context(), config, prometheus.NewRegistry())
	require.NoError(t, err)
	_, err = first.register("google.protobuf.StringValue", "renamed")
	require.NoError(t, err)
	require.NoError(t, first.seal(t.Context()))
	require.NoError(t, first.close())

	second, err := newEventBus(t.Context(), config, prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.close() })
	_, err = second.register("google.protobuf.StringValue", "player")
	require.NoError(t, err)
	require.NoError(t, second.seal(t.Context()))

	groups, err := second.client.XInfoGroups(t.Context(), stream).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "player", groups[0].Name)
}

func TestAPublishNobodyHearsAddsNothing(t *testing.T) {
	bus, _ := sealedBus(t, "player")

	bus.Publish(wrapperspb.Int64(7))

	exists, err := bus.client.Exists(t.Context(), streamOf("google.protobuf.Int64Value")).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}
