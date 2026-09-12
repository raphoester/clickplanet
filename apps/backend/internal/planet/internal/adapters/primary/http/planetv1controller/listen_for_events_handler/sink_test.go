package listen_for_events_handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events"
)

type recorder struct {
	sent []*planetv1.PlanetEvent
	err  error
}

func (r *recorder) Send(event *planetv1.PlanetEvent) error {
	r.sent = append(r.sent, event)
	return r.err
}

func TestSinkFramesATileUpdate(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Update: clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)

	update := stream.sent[0].GetTileUpdate()
	require.NotNil(t, update, "a tile update travels as the tile_update case")
	assert.Nil(t, stream.sent[0].GetHeartbeat())
	assert.Equal(t, uint32(42), update.GetTileId())
	assert.Equal(t, "fr", update.GetCountryId())
	assert.Equal(t, "de", update.GetPreviousCountryId())
}

func TestSinkFramesAHeartbeat(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{Heartbeat: true})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	require.NotNil(t, stream.sent[0].GetHeartbeat(), "a heartbeat travels as the heartbeat case")
	assert.Nil(t, stream.sent[0].GetTileUpdate())
}

// A stream that has gone away has to end the feed, not be retried silently.
func TestSinkReportsAFailedSend(t *testing.T) {
	stream := &recorder{err: assert.AnError}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{Heartbeat: true})

	require.ErrorIs(t, err, assert.AnError)
}
