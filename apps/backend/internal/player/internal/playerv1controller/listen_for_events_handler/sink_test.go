package listen_for_events_handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

type recordingStream struct {
	sent []*playerv1.PlayerEvent
}

func (r *recordingStream) Send(event *playerv1.PlayerEvent) error {
	r.sent = append(r.sent, event)
	return nil
}

var ada = presence.Entry{Key: "k1", Name: "Ada_L", Country: "fr"}

func TestEachFrameIsItsCaseOfTheEnvelope(t *testing.T) {
	stream := &recordingStream{}
	sink := listen_for_events_handler.NewSink(stream)

	require.NoError(t, sink.SendRoster([]presence.Entry{ada}))
	require.NoError(t, sink.SendChange(presence.Change{Entry: ada}))
	require.NoError(t, sink.SendChange(presence.Change{Entry: ada, Left: true}))
	require.NoError(t, sink.SendHeartbeat())

	entry := &playerv1.RosterEntry{Key: "k1", Name: "Ada_L", CountryId: "fr"}
	want := []*playerv1.PlayerEvent{
		{Event: &playerv1.PlayerEvent_Roster{Roster: &playerv1.Roster{Entries: []*playerv1.RosterEntry{entry}}}},
		{Event: &playerv1.PlayerEvent_Entry{Entry: entry}},
		{Event: &playerv1.PlayerEvent_Left{Left: &playerv1.PlayerLeft{Key: "k1"}}},
		{Event: &playerv1.PlayerEvent_Heartbeat{Heartbeat: &playerv1.Heartbeat{}}},
	}
	require.Len(t, stream.sent, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], stream.sent[i]), "frame %d: %v", i, stream.sent[i])
	}
}
