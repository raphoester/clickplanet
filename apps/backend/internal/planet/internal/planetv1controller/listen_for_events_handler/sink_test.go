package listen_for_events_handler_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type recorder struct {
	sent []*planetv1.PlanetEvent
}

func (r *recorder) Send(event *planetv1.PlanetEvent) error {
	r.sent = append(r.sent, event)
	return nil
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

func TestSinkFramesAChangeOfRules(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{RulesChanged: true})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	assert.NotNil(t, stream.sent[0].GetRulesChanged())
	assert.Nil(t, stream.sent[0].GetTileUpdate())
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
	require.Len(t, dropped.GetStruck(), 1)
	assert.Equal(t, uint32(9), dropped.GetStruck()[0].GetTileId())
	assert.Equal(t, uint32(2), dropped.GetStruck()[0].GetShields(), "with the shields it kept")
	assert.InDelta(t, 1, dropped.GetPoint().GetZ(), 1e-9)
}

func TestASinkFramesAFortifiedProvince(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Fortification: &clicks.Fortification{Province: 42, Country: "us", Tile: 1234},
	}))

	fortified := stream.sent[0].GetProvinceFortified()
	require.NotNil(t, fortified, "expected a province_fortified case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, uint32(42), fortified.GetProvinceId())
	assert.Equal(t, "us", fortified.GetCountryId())
	assert.Equal(t, uint32(1234), fortified.GetTileId())
}

func TestASinkFramesAnEnclosureAsOneEventWithItsTilesAndItsVersion(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Version: clicks.MapVersion{Epoch: 7, Seq: 42},
		Enclosure: &clicks.Enclosure{
			Country: "jp", ClosingTile: 4, Wall: []uint32{4, 5, 6}, Filled: []uint32{9, 10},
			Struck: []clicks.TileShields{{Tile: 10, Shields: 3}},
		},
		Yours: true,
	}))

	require.Len(t, stream.sent, 1)
	enclosed := stream.sent[0].GetTilesEnclosed()
	require.NotNil(t, enclosed)
	assert.Equal(t, "jp", enclosed.GetCountryId())
	assert.Equal(t, uint32(4), enclosed.GetClosingTileId())
	assert.Equal(t, []uint32{4, 5, 6}, enclosed.GetWallTileIds())
	assert.Equal(t, []uint32{9, 10}, enclosed.GetFilledTileIds())
	require.Len(t, enclosed.GetStruck(), 1)
	assert.Equal(t, uint32(10), enclosed.GetStruck()[0].GetTileId())
	assert.Equal(t, uint32(3), enclosed.GetStruck()[0].GetShields())
	assert.Equal(t, uint64(42), enclosed.GetVersion().GetSeq())
	assert.True(t, enclosed.GetYours())
}

func TestASinkFramesASpreadClick(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Spread: &notices.Spread{CountryID: "br", Tile: 100, Neighbours: []uint32{99, 101}},
	}))

	spread := stream.sent[0].GetTilesSpread()
	require.NotNil(t, spread)
	assert.Equal(t, "br", spread.GetCountryId())
	assert.Equal(t, uint32(100), spread.GetTileId())
	assert.Equal(t, []uint32{99, 101}, spread.GetSpreadTileIds())
}

func TestASinkFramesWhatAGrantAddedAndEverythingItLeftHeld(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Granted: &charges.Gain{
			Paid:   []charges.Kind{charges.KindBomb, charges.KindSpreadClicks, charges.KindShields},
			Before: charges.Held{Enclosures: 2, Shields: 5},
			After:  charges.Held{Bombs: 1, Enclosures: 2, SpreadClicks: 4, Shields: 20},
		},
	}))

	granted := stream.sent[0].GetChargesGranted()
	held := granted.GetCharges()
	require.NotNil(t, held)
	assert.Equal(t, uint32(1), held.GetBombs())
	assert.Zero(t, held.GetRefills())
	assert.Equal(t, uint32(2), held.GetEnclosures())
	assert.Equal(t, uint32(4), held.GetSpreadClicksLeft())
	assert.Equal(t, uint32(20), held.GetShields())

	gained := map[planetv1.BonusKind]uint32{}
	for _, each := range granted.GetGained() {
		gained[each.GetKind()] = each.GetAmount()
	}
	assert.Equal(t, map[planetv1.BonusKind]uint32{
		planetv1.BonusKind_BONUS_KIND_BOMB: 1, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS: 4, planetv1.BonusKind_BONUS_KIND_SHIELDS: 15,
	}, gained, "the enclosures it held before are no gain")
}

func TestASinkFramesAKindPaidIntoAFullStackAsNoneKept(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Granted: &charges.Gain{
			Paid:   []charges.Kind{charges.KindBomb},
			Before: charges.Held{Bombs: 1},
			After:  charges.Held{Bombs: 1},
		},
	}))

	gained := stream.sent[0].GetChargesGranted().GetGained()
	require.Len(t, gained, 1)
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_BOMB, gained[0].GetKind())
	assert.Zero(t, gained[0].GetAmount())
}

func TestASinkFramesAKnockOutAsOneEventWithItsTilesAndVersion(t *testing.T) {
	stream := &recorder{}

	err := listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Version:  clicks.MapVersion{Epoch: 1_790_000_000, Seq: 9},
		KnockOut: &clicks.KnockOut{Country: "fr", By: "de", Tile: 4, Cleared: []uint32{5, 6}},
	})

	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	assert.True(t, proto.Equal(&planetv1.KnockOut{
		CountryId: "fr", ByCountryId: "de", TileId: 4, ClearedTileIds: []uint32{5, 6},
		Version: &planetv1.MapVersion{Epoch: 1_790_000_000, Seq: 9},
	}, stream.sent[0].GetKnockOut()))
}

func TestASinkFramesAVirusSpreadAsOneEventWithItsTilesAndItsVersion(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Version: clicks.MapVersion{Epoch: 7, Seq: 43},
		Infection: &clicks.Infection{
			Virus:  clicks.VirusSnapshot{ID: "v-1", Flag: "fr", Victim: "de", Tile: 5, Infected: []uint32{5, 4}, Ended: true},
			Taken:  []uint32{4},
			Struck: []clicks.TileShields{{Tile: 6, Shields: 2}},
		},
	}))

	require.Len(t, stream.sent, 1)
	spread := stream.sent[0].GetVirusSpread()
	require.NotNil(t, spread)
	assert.Equal(t, "v-1", spread.GetVirusId())
	assert.Equal(t, "fr", spread.GetCountryId())
	assert.Equal(t, "de", spread.GetVictimCountryId())
	assert.Equal(t, []uint32{4}, spread.GetTakenTileIds())
	require.Len(t, spread.GetStruck(), 1)
	assert.Equal(t, uint32(2), spread.GetStruck()[0].GetShields())
	assert.True(t, spread.GetEnded())
	assert.Equal(t, uint64(43), spread.GetVersion().GetSeq())
}

func TestASinkFramesAnExplosionAsOneEventWithEachBlastInOrder(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Version: clicks.MapVersion{Epoch: 7, Seq: 44},
		Explosion: &clicks.Explosion{Blasts: []clicks.MineBlast{
			{
				Mine: clicks.MineSnapshot{Tile: 5, Flag: "fr"}, Taken: []uint32{5, 4},
				Struck: []clicks.TileShields{{Tile: 6, Shields: 2}},
			},
			{Mine: clicks.MineSnapshot{Tile: 9, Flag: "de"}},
		}},
	}))

	require.Len(t, stream.sent, 1)
	assert.True(t, proto.Equal(&planetv1.MinesExploded{
		Blasts: []*planetv1.MineBlast{
			{
				TileId: 5, CountryId: "fr", TakenTileIds: []uint32{5, 4},
				Struck: []*planetv1.TileShields{{TileId: 6, Shields: 2}},
			},
			{TileId: 9, CountryId: "de"},
		},
		Version: &planetv1.MapVersion{Epoch: 7, Seq: 44},
	}, stream.sent[0].GetMinesExploded()), "%v", stream.sent[0])
}

func TestASinkFramesAMineLaidWithTheVersionItWasLaidAt(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		MineLaid: &notices.MineLaid{Flag: "fr", Tile: 9, Version: clicks.MapVersion{Epoch: 7, Seq: 3}},
	}))

	require.Len(t, stream.sent, 1)
	assert.True(t, proto.Equal(&planetv1.MineLaid{
		CountryId: "fr", TileId: 9, Version: &planetv1.MapVersion{Epoch: 7, Seq: 3},
	}, stream.sent[0].GetMineLaid()), "%v", stream.sent[0])
}

func TestASinkFramesARallyWithTheTimeItHasLeft(t *testing.T) {
	stream := &recorder{}
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	rally := rallies.Snapshot{
		ID: rallies.RallyID(uuid.MustParse("00000000-0000-7000-8000-000000000001")), Flag: "fr", Centre: clicks.Vec3{X: 0.6, Z: 0.8}, Radius: 0.072,
		Account:  cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")),
		CalledAt: at, EndsAt: at.Add(3 * time.Minute),
	}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Rally: &notices.RallyCalled{Rally: rally, Left: 3 * time.Minute},
	}))

	require.Len(t, stream.sent, 1)
	assert.True(t, proto.Equal(&planetv1.Rally{
		RallyId: "00000000-0000-7000-8000-000000000001", AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", CountryId: "fr",
		Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}, Radius: 0.072, LeftMs: 180_000,
	}, stream.sent[0].GetRallyCalled()), "%v", stream.sent[0])
}
