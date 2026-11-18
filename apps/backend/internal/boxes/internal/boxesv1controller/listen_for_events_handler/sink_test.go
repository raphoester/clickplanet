package listen_for_events_handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/listen_for_events_handler"
)

type recorder struct {
	sent []*boxesv1.BoxEvent
}

func (r *recorder) Send(event *boxesv1.BoxEvent) error {
	r.sent = append(r.sent, event)
	return nil
}

func TestASinkFramesABoxOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Offer: &boxes.Offer{Token: "a-token", Seed: 42, ExpiresAt: time.Unix(0, 0).Add(15 * time.Second)},
	}))

	offered := stream.sent[0].GetBoxOffered()
	require.NotNil(t, offered, "expected a box_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-token", offered.GetToken())
	assert.Equal(t, uint32(42), offered.GetSeed())
	assert.Equal(t, int64(15_000), offered.GetExpiresAtUnixMs())
}

func TestASinkFramesAHeartbeat(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{Heartbeat: true}))

	require.Len(t, stream.sent, 1)
	assert.NotNil(t, stream.sent[0].GetHeartbeat())
}

func TestASinkSendsNothingForAnEmptyEvent(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{}))
	assert.Empty(t, stream.sent)
}
