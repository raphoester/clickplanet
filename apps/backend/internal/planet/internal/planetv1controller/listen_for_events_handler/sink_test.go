package listen_for_events_handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
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

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Version: clicks.MapVersion{Epoch: 1_790_000_000, Seq: 7},
		Update:  &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)

	update := stream.sent[0].GetTileUpdate()
	require.NotNil(t, update, "a tile update travels as the tile_update case")
	assert.Nil(t, stream.sent[0].GetHeartbeat())
	assert.Equal(t, uint32(42), update.GetTileId())
	assert.Equal(t, "fr", update.GetCountryId())
	assert.Equal(t, "de", update.GetPreviousCountryId())
	assert.False(t, update.GetClicked())
	assert.Equal(t, uint32(1_790_000_000), update.GetVersion().GetEpoch())
	assert.Equal(t, uint64(7), update.GetVersion().GetSeq())
}

func TestSinkSaysWhichUpdateIsAClick(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Clicked: true},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	assert.True(t, stream.sent[0].GetTileUpdate().GetClicked())
}

func TestSinkSaysHowManyShieldsAreLeft(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "fr", Shields: 9},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	assert.Equal(t, uint32(9), stream.sent[0].GetTileUpdate().GetShields())
}

func TestSinkFramesAHeartbeat(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Heartbeat: true, Version: clicks.MapVersion{Epoch: 1_790_000_000, Seq: 3},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	require.NotNil(t, stream.sent[0].GetHeartbeat(), "a heartbeat travels as the heartbeat case")
	assert.Nil(t, stream.sent[0].GetTileUpdate())
	assert.Equal(t, uint64(3), stream.sent[0].GetHeartbeat().GetVersion().GetSeq())
}

func TestSinkRefusesAnEventWithNothingInIt(t *testing.T) {
	stream := &recorder{}

	require.Error(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{}))
	assert.Empty(t, stream.sent)
}

func TestASinkFramesTheHoldingsThatChanged(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Version: clicks.MapVersion{Epoch: 1_790_000_000, Seq: 12},
		Tally: &clicks.Tally{
			Countries:  []clicks.FlagTiles{{Flag: "fr", Tiles: 3}, {Flag: "de"}},
			Landmasses: []clicks.LandmassTiles{{Landmass: 4, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 3}}}, {Landmass: 5}},
		},
	}))

	assert.True(t, proto.Equal(&planetv1.HoldingsChanged{
		Version:   &planetv1.MapVersion{Epoch: 1_790_000_000, Seq: 12},
		Countries: []*planetv1.Holding{{CountryId: "fr", Tiles: 3}, {CountryId: "de"}},
		Landmasses: []*planetv1.LandmassHoldings{
			{LandmassId: 4, Holdings: []*planetv1.Holding{{CountryId: "fr", Tiles: 3}}},
			{LandmassId: 5},
		},
	}, stream.sent[0].GetHoldingsChanged()), "a flag at zero and a landmass nobody holds are listed: %v", stream.sent[0])
}

func TestSinkReportsAFailedSend(t *testing.T) {
	stream := &recorder{err: assert.AnError}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{Heartbeat: true})

	require.ErrorIs(t, err, assert.AnError)
}

func TestASinkFramesABoxOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Offer: &bonuses.Offer{
			Token:     "a-token",
			Seed:      42,
			Kind:      bonuses.KindSpreadClicks,
			ExpiresAt: time.Unix(0, 0).Add(15 * time.Second),
		},
	}))

	offered := stream.sent[0].GetBonusOffered()
	require.NotNil(t, offered, "expected a bonus_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-token", offered.GetToken())
	assert.Equal(t, uint32(42), offered.GetSeed())
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, offered.GetKind())
	assert.Equal(t, int64(15_000), offered.GetExpiresAtUnixMs())
}

func TestASinkFramesAQuizOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Quiz: &bonuses.QuizOffer{Token: "a-quiz", ExpiresAt: time.Unix(0, 0).Add(20 * time.Second)},
	}))

	quiz := stream.sent[0].GetQuizOffered()
	require.NotNil(t, quiz, "expected a quiz_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-quiz", quiz.GetToken())
	assert.Equal(t, int64(20_000), quiz.GetExpiresAtUnixMs())
}

func TestASinkFramesACatchWithNoTokenOnIt(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Taken: &bonuses.Taken{CountryID: "jp", Kind: bonuses.KindRefill},
	}))

	taken := stream.sent[0].GetBonusTaken()
	require.NotNil(t, taken)
	assert.Equal(t, "jp", taken.GetCountryId())
}

func TestASinkFramesABlastWithEveryTileItCleared(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Blast: &clicks.Blast{
			Tile: 7, CountryID: "fr", Radius: 0.03,
			Point:   clicks.Vec3{X: 0, Y: 0, Z: 1},
			Cleared: []uint32{6, 7, 8},
			Struck:  []uint32{9},
			Left:    []int{2},
		},
	}))

	dropped := stream.sent[0].GetBombDropped()
	require.NotNil(t, dropped, "expected a bomb_dropped case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, uint32(7), dropped.GetTileId())
	assert.Equal(t, "fr", dropped.GetCountryId())
	assert.InDelta(t, 0.03, dropped.GetRadius(), 1e-9)
	assert.Equal(t, []uint32{6, 7, 8}, dropped.GetClearedTileIds())
	assert.Equal(t, []uint32{9}, dropped.GetStruckTileIds()) //nolint:staticcheck // still sent to the tabs loaded before it was deprecated.
	require.Len(t, dropped.GetStruck(), 1)
	assert.Equal(t, uint32(9), dropped.GetStruck()[0].GetTileId())
	assert.Equal(t, uint32(2), dropped.GetStruck()[0].GetShields(), "with the shields it kept")
	assert.InDelta(t, 1, dropped.GetPoint().GetZ(), 1e-9)
}

func TestASinkFramesAFortifiedLandmass(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Fortification: &clicks.Fortification{Landmass: 42, Country: "us", Tile: 1234},
	}))

	fortified := stream.sent[0].GetLandmassFortified()
	require.NotNil(t, fortified, "expected a landmass_fortified case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, uint32(42), fortified.GetLandmassId())
	assert.Equal(t, "us", fortified.GetCountryId())
	assert.Equal(t, uint32(1234), fortified.GetTileId())
}

func TestASinkFramesAClosedShape(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Enclosed: &bonuses.Enclosed{
			CountryID: "jp", ClosingTile: 4, Wall: []uint32{4, 5, 6}, Filled: []uint32{9, 10},
			Yours: true,
		},
	}))

	enclosed := stream.sent[0].GetTilesEnclosed()
	require.NotNil(t, enclosed)
	assert.Equal(t, "jp", enclosed.GetCountryId())
	assert.Equal(t, uint32(4), enclosed.GetClosingTileId())
	assert.Equal(t, []uint32{4, 5, 6}, enclosed.GetWallTileIds())
	assert.Equal(t, []uint32{9, 10}, enclosed.GetFilledTileIds())
	assert.True(t, enclosed.GetYours())
}

func TestASinkFramesASpreadClick(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Spread: &bonuses.Spread{CountryID: "br", Tile: 100, Neighbours: []uint32{99, 101}},
	}))

	spread := stream.sent[0].GetTilesSpread()
	require.NotNil(t, spread)
	assert.Equal(t, "br", spread.GetCountryId())
	assert.Equal(t, uint32(100), spread.GetTileId())
	assert.Equal(t, []uint32{99, 101}, spread.GetSpreadTileIds())
}

func TestAHeartbeatStillWinsOverEverything(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{Heartbeat: true}))

	assert.NotNil(t, stream.sent[0].GetHeartbeat())
}
