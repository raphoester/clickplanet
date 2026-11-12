package cpredis_test

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// BytesValue and StringValue have one field, number 1, of the same wire type: the one reads as the other, as an event
// that moved package with its fields does.
var oldName = (&wrapperspb.BytesValue{}).ProtoReflect().Descriptor().FullName()

func movedSubscriber(
	t *testing.T, streams *cpredis.Streams, from protoreflect.FullName, name string, handler cpredis.Handler[*wrapperspb.StringValue],
) cpbootstrap.Runner {
	t.Helper()

	runner, err := cpredis.SubscribeMoved(t.Context(), streams, from, name, handler)
	require.NoError(t, err)
	return runner
}

func publishOld(streams *cpredis.Streams, values ...string) {
	events := make([]proto.Message, len(values))
	for i, value := range values {
		events[i] = wrapperspb.Bytes([]byte(value))
	}
	streams.Publish(events...)
}

func TestWhatTheOldStreamHeldIsHandledBeforeTheNewOne(t *testing.T) {
	server.Flush(t)
	config := server.Config()
	before := connected(t, config, prometheus.NewRegistry())
	// The group the module read the old stream with, as a deploy leaves it: never run again.
	subscribed(t, before, "missions-boxes",
		handlerFunc[*wrapperspb.BytesValue](func(context.Context, *wrapperspb.BytesValue) error { return nil }))
	publishOld(before, "told before", "and before")

	after := connected(t, config, prometheus.NewRegistry())
	heard := &received{}
	publish(after, "told since")
	stop := running(movedSubscriber(t, after, oldName, "missions-boxes", heard))
	eventually(t, func() bool { return len(heard.values()) == 3 })
	stop()

	assert.Equal(t, []string{"told before", "and before", "told since"}, heard.values())
}

func TestWhatTheOldGroupHandledIsNotHandledAgain(t *testing.T) {
	server.Flush(t)
	config := server.Config()
	before := connected(t, config, prometheus.NewRegistry())
	old := &received{}
	oldDone := make(chan struct{}, 1)
	stopOld := running(subscribed(t, before, "missions-boxes",
		handlerFunc[*wrapperspb.BytesValue](func(ctx context.Context, event *wrapperspb.BytesValue) error {
			oldDone <- struct{}{}
			return old.Handle(ctx, wrapperspb.String(string(event.GetValue())))
		})))
	publishOld(before, "handled before")
	<-oldDone
	stopOld()
	publishOld(before, "left")

	after := connected(t, config, prometheus.NewRegistry())
	heard := &received{}
	publish(after, "told since")
	stop := running(movedSubscriber(t, after, oldName, "missions-boxes", heard))
	eventually(t, func() bool { return len(heard.values()) == 2 })
	stop()

	assert.Equal(t, []string{"left", "told since"}, heard.values(), "counted once, in the order they were told")
}

func TestAGroupThatNeverReadTheOldStreamStartsOnTheNewOne(t *testing.T) {
	server.Flush(t)
	config := server.Config()
	streams := connected(t, config, prometheus.NewRegistry())
	publishOld(streams, "another module's")
	heard := &received{}

	stop := running(movedSubscriber(t, streams, oldName, "fresh", heard))
	publish(streams, "told since")
	eventually(t, func() bool { return len(heard.values()) == 1 })
	stop()

	assert.Equal(t, []string{"told since"}, heard.values())
}

func TestNoOldStreamIsNothingLeft(t *testing.T) {
	server.Flush(t)
	streams := connected(t, server.Config(), prometheus.NewRegistry())
	heard := &received{}

	stop := running(movedSubscriber(t, streams, "gone.v1.Event", "fresh", heard))
	publish(streams, "told since")
	eventually(t, func() bool { return len(heard.values()) == 1 })
	stop()

	assert.Equal(t, []string{"told since"}, heard.values())
}
