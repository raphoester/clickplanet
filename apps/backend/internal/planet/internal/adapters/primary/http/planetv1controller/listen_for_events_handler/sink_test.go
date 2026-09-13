package listen_for_events_handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
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

func TestASinkFramesABoxOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Offer: &bonus.Offer{
			Token:     "a-token",
			Seed:      42,
			Kind:      bonus.KindTripleClicks,
			Duration:  time.Minute,
			ExpiresAt: time.Unix(0, 0).Add(15 * time.Second),
		},
	}))

	offered := stream.sent[0].GetBonusOffered()
	require.NotNil(t, offered, "expected a bonus_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-token", offered.GetToken())
	assert.Equal(t, uint32(42), offered.GetSeed())
	assert.Equal(t, uint32(60), offered.GetDurationSeconds())
	assert.Equal(t, int64(15_000), offered.GetExpiresAtUnixMs())
}

func TestASinkFramesACatchWithNoTokenOnIt(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Taken: &bonus.Taken{CountryID: "jp", Kind: bonus.KindTripleClicks},
	}))

	taken := stream.sent[0].GetBonusTaken()
	require.NotNil(t, taken)
	assert.Equal(t, "jp", taken.GetCountryId())
}

func TestASinkFramesABlastWithEveryTileItCleared(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Blast: &clicks.Blast{
			Tile: 7, CountryID: "fr", Radius: 0.03,
			Point:   clicks.Vec3{X: 0, Y: 0, Z: 1},
			Cleared: []uint32{6, 7, 8},
		},
	}))

	dropped := stream.sent[0].GetBombDropped()
	require.NotNil(t, dropped, "expected a bomb_dropped case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, uint32(7), dropped.GetTileId())
	assert.Equal(t, "fr", dropped.GetCountryId())
	assert.InDelta(t, 0.03, dropped.GetRadius(), 1e-9)
	assert.Equal(t, []uint32{6, 7, 8}, dropped.GetClearedTileIds())
	assert.InDelta(t, 1, dropped.GetPoint().GetZ(), 1e-9)
}

func TestASinkFramesAClosedShape(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Enclosed: &bonus.Enclosed{
			CountryID: "jp", ClosingTile: 4, Wall: []uint32{4, 5, 6}, Filled: []uint32{9, 10},
			Yours: true, Left: 2,
		},
	}))

	enclosed := stream.sent[0].GetTilesEnclosed()
	require.NotNil(t, enclosed)
	assert.Equal(t, "jp", enclosed.GetCountryId())
	assert.Equal(t, uint32(4), enclosed.GetClosingTileId())
	assert.Equal(t, []uint32{4, 5, 6}, enclosed.GetWallTileIds())
	assert.Equal(t, []uint32{9, 10}, enclosed.GetFilledTileIds())
	assert.True(t, enclosed.GetYours())
	assert.Equal(t, uint32(2), enclosed.GetEnclosuresLeft())
}

func TestASinkFramesASpreadClick(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Spread: &bonus.Spread{CountryID: "br", Tile: 100, Neighbours: []uint32{99, 101}},
	}))

	spread := stream.sent[0].GetTilesSpread()
	require.NotNil(t, spread)
	assert.Equal(t, "br", spread.GetCountryId())
	assert.Equal(t, uint32(100), spread.GetTileId())
	assert.Equal(t, []uint32{99, 101}, spread.GetSpreadTileIds())
}

func TestASinkSaysATileUpdateWasBoosted(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{
		Update: clicks.TileUpdate{Tile: 42, Value: "it", Boosted: true},
	}))

	update := stream.sent[0].GetTileUpdate()
	require.NotNil(t, update)
	assert.True(t, update.GetBoosted())
}

func TestAHeartbeatStillWinsOverEverything(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events.Event{Heartbeat: true}))

	assert.NotNil(t, stream.sent[0].GetHeartbeat())
}
