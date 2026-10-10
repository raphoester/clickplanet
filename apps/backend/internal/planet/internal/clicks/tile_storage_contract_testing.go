//go:build testing

package clicks

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/stretchr/testify/suite"
)

const contractMaxIndex = 100_000

var (
	contractIsland = []uint32{90_001, 90_002, 90_003}
	contractIslet  = []uint32{90_004}
)

type TileStorageContractSuite struct {
	suite.Suite

	NewStorage func(borders *Borders) TileStorage

	storage TileStorage
}

func (s *TileStorageContractSuite) SetupTest() {
	s.storage = s.NewStorage(BordersOf(contractMaxIndex, contractIsland, contractIslet))
}

func (s *TileStorageContractSuite) subscribe(timeout time.Duration) (<-chan Change, context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s.T().Cleanup(cancel)

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	return listener, ctx
}

func (s *TileStorageContractSuite) next(ctx context.Context, listener <-chan Change) Change {
	select {
	case <-ctx.Done():
		s.T().Fatal("timeout")
	case change := <-listener:
		return change
	}

	return Change{}
}

func (s *TileStorageContractSuite) owners(start, end uint32) map[uint32]string {
	batch, err := s.storage.StateBatchDense(start, end)
	s.Require().NoError(err)

	state := make(map[uint32]string)
	for i := 0; i+1 < len(batch.Tiles); i += 2 {
		if code := binary.LittleEndian.Uint16(batch.Tiles[i : i+2]); code != 0 {
			state[batch.Start+uint32(i/2)] = batch.Codes[code] //nolint:gosec // i/2 is a tile offset inside the batch.
		}
	}

	return state
}

func (s *TileStorageContractSuite) TestSetPublishesTheUpdate() {
	listener, ctx := s.subscribe(2 * time.Second)

	s.Require().NoError(s.storage.Set(ctx, 10, "fr"))

	s.Equal(&TileUpdate{Tile: 10, Value: "fr"}, s.next(ctx, listener).Update)
}

func (s *TileStorageContractSuite) TestClickPublishesTheUpdateAsAClick() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "us"))
	listener, ctx := s.subscribe(2 * time.Second)

	s.Require().NoError(s.storage.Click(ctx, 10, "fr"))

	s.Equal(&TileUpdate{Tile: 10, Value: "fr", Previous: "us", Clicked: true}, s.next(ctx, listener).Update)
}

func (s *TileStorageContractSuite) TestClickOnATileAlreadyHeldPublishesNothing() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))
	listener, ctx := s.subscribe(200 * time.Millisecond)

	s.Require().NoError(s.storage.Click(context.Background(), 10, "fr"))

	select {
	case <-ctx.Done():
	case change, open := <-listener:
		if open {
			s.T().Errorf("unexpected change %v", change)
		}
	}
}

func (s *TileStorageContractSuite) TestClickRefusesATileOutOfRange() {
	s.Error(s.storage.Click(context.Background(), contractMaxIndex+1, "fr"))
}

func (s *TileStorageContractSuite) TestSetOverATileCarriesThePreviousOwner() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "us"))
	listener, ctx := s.subscribe(2 * time.Second)

	s.Require().NoError(s.storage.Set(ctx, 10, "fr"))

	s.Equal(&TileUpdate{Tile: 10, Value: "fr", Previous: "us"}, s.next(ctx, listener).Update)
}

func (s *TileStorageContractSuite) TestSetOnATileAlreadyHeldPublishesNothing() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))
	listener, ctx := s.subscribe(200 * time.Millisecond)

	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))

	select {
	case <-ctx.Done():
	case change, open := <-listener:
		if open {
			s.T().Errorf("unexpected change %v", change)
		}
	}
}

func (s *TileStorageContractSuite) TestSetRefusesATileOutOfRange() {
	s.Error(s.storage.Set(context.Background(), contractMaxIndex+1, "fr"))
}

func (s *TileStorageContractSuite) TestEverySubscriberGetsEveryChange() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listeners := make([]<-chan Change, 0, 5)
	for range 5 {
		listener, err := s.storage.Subscribe(ctx)
		s.Require().NoError(err)
		listeners = append(listeners, listener)
	}

	s.Require().NoError(s.storage.Set(ctx, 42, "fr"))

	for _, listener := range listeners {
		s.Equal(&TileUpdate{Tile: 42, Value: "fr"}, s.next(ctx, listener).Update)
	}
}

func (s *TileStorageContractSuite) TestSubscribeClosesTheFeedWhenTheContextEnds() {
	ctx, cancel := context.WithCancel(context.Background())

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	cancel()

	select {
	case _, open := <-listener:
		s.False(open, "the feed is closed")
	case <-time.After(2 * time.Second):
		s.T().Fatal("the feed was not closed after the context was cancelled")
	}

	s.NoError(s.storage.Set(context.Background(), 1, "fr"))
}

func (s *TileStorageContractSuite) TestClearEmptiesTheTilesAndPublishesOneBlast() {
	s.Require().NoError(s.storage.Set(context.Background(), 10, "fr"))
	s.Require().NoError(s.storage.Set(context.Background(), 12, "jp"))
	listener, ctx := s.subscribe(2 * time.Second)

	blast, err := s.storage.Clear(ctx, Blast{Tile: 11, CountryID: "de", Cleared: []uint32{10, 11, 12}})
	s.Require().NoError(err)
	s.Equal([]uint32{10, 12}, blast.Cleared, "only tiles that were held are reported")
	s.Equal([]string{"fr", "jp"}, blast.Owners, "with the flag each one wore")

	change := s.next(ctx, listener)
	s.Require().NotNil(change.Blast)
	s.Equal([]uint32{10, 12}, change.Blast.Cleared)
	s.Equal("de", change.Blast.CountryID)
	s.Empty(listener, "one event for the blast, not one per tile")

	owner, _ := s.storage.Owner(10)
	s.Empty(owner)
}

func (s *TileStorageContractSuite) TestClearRefusesATileOutOfRange() {
	_, err := s.storage.Clear(context.Background(), Blast{Cleared: []uint32{contractMaxIndex + 1}})
	s.Error(err)
}

func (s *TileStorageContractSuite) TestAStateBatchReadsTheHeldTilesInRange() {
	for _, tile := range []uint32{10, 20, 30} {
		s.Require().NoError(s.storage.Set(context.Background(), tile, "fr"))
	}

	s.Equal(map[uint32]string{10: "fr", 20: "fr", 30: "fr"}, s.owners(10, 30))
	s.Equal(map[uint32]string{10: "fr", 20: "fr", 30: "fr"}, s.owners(5, contractMaxIndex+1_000), "an end past the map is clamped")
}

func (s *TileStorageContractSuite) TestAStateBatchIsDenseAndKeepsItsOffset() {
	storage := s.NewStorage(BordersOf(9))
	s.Require().NoError(storage.Set(context.Background(), 2, "fr"))
	s.Require().NoError(storage.Set(context.Background(), 4, "gb-eng"))

	read := func(start, end uint32) (uint32, []string) {
		batch, err := storage.StateBatchDense(start, end)
		s.Require().NoError(err)

		codes := make([]string, 0, len(batch.Tiles)/2)
		for i := 0; i+1 < len(batch.Tiles); i += 2 {
			codes = append(codes, batch.Codes[binary.LittleEndian.Uint16(batch.Tiles[i:i+2])])
		}

		return batch.Start, codes
	}

	start, codes := read(0, 9)
	s.Equal(uint32(0), start)
	s.Equal([]string{"", "", "fr", "", "gb-eng", "", "", "", "", ""}, codes, "unowned tiles read as empty")

	start, codes = read(4, 5)
	s.Equal(uint32(4), start)
	s.Equal([]string{"gb-eng", ""}, codes)

	batch, err := storage.StateBatchDense(0, 0)
	s.Require().NoError(err)
	batch.Codes[0] = "tampered"
	_, codes = read(0, 0)
	s.Equal([]string{""}, codes, "the code table is a copy, not the live one")

	_, err = storage.StateBatchDense(5, 4)
	s.Error(err, "an inverted range is refused")
}

func (s *TileStorageContractSuite) TestShareFollowsSetsAndBlasts() {
	ctx := context.Background()
	for tile := uint32(1); tile <= 10; tile++ {
		s.Require().NoError(s.storage.Set(ctx, tile, "bg"))
	}
	s.Require().NoError(s.storage.Set(ctx, 3, "fr"))
	s.Require().NoError(s.storage.Set(ctx, 4, "bg"))

	s.InDelta(9.0/contractMaxIndex, s.storage.Share("bg"), 1e-12)
	s.InDelta(1.0/contractMaxIndex, s.storage.Share("fr"), 1e-12)

	_, err := s.storage.Clear(ctx, Blast{Cleared: []uint32{1, 2, 3, 50}})
	s.Require().NoError(err)

	s.InDelta(7.0/contractMaxIndex, s.storage.Share("bg"), 1e-12)
	s.Zero(s.storage.Share("fr"))
	s.Zero(s.storage.Share("de"), "a country that never clicked holds nothing")
	s.Zero(s.storage.Share(""), "unowned ground is nobody's share")
}

func (s *TileStorageContractSuite) TestReassignMovesOnlyTheTilesFromHolds() {
	ctx := context.Background()
	for tile, country := range map[uint32]string{3: "dz", 4: "bg", 5: "dz", 9: "fr"} {
		s.Require().NoError(s.storage.Set(ctx, tile, country))
	}

	next, moved, err := s.storage.Reassign(ctx, "dz", "fr", 0, 100)
	s.Require().NoError(err)

	s.Equal(uint32(0), next, "the scan reached the end of the map")
	s.Equal(2, moved)
	s.Equal(map[uint32]string{3: "fr", 4: "bg", 5: "fr", 9: "fr"}, s.owners(0, 10))
	s.Equal(0, s.storage.Held("dz"))
	s.Equal(3, s.storage.Held("fr"))
	s.InDelta(3.0/contractMaxIndex, s.storage.Share("fr"), 1e-12, "the toll reads the same counts")
}

func (s *TileStorageContractSuite) TestReassignPublishesOneOrdinaryUpdatePerTile() {
	s.Require().NoError(s.storage.Set(context.Background(), 7, "dz"))
	s.Require().NoError(s.storage.Set(context.Background(), 8, "dz"))
	listener, ctx := s.subscribe(2 * time.Second)

	_, _, err := s.storage.Reassign(ctx, "dz", "fr", 0, 100)
	s.Require().NoError(err)

	for _, tile := range []uint32{7, 8} {
		s.Equal(&TileUpdate{Tile: tile, Value: "fr", Previous: "dz"}, s.next(ctx, listener).Update)
	}
	s.Empty(listener)
}

func (s *TileStorageContractSuite) TestReassignStopsAtTheLimitAndResumesWhereItStopped() {
	ctx := context.Background()
	for _, tile := range []uint32{10, 20, 30, 40, 50} {
		s.Require().NoError(s.storage.Set(ctx, tile, "dz"))
	}

	next, moved, err := s.storage.Reassign(ctx, "dz", "fr", 0, 2)
	s.Require().NoError(err)
	s.Equal(2, moved)
	s.Equal(uint32(30), next, "the next tile still held, not the one after the last moved")

	next, moved, err = s.storage.Reassign(ctx, "dz", "fr", next, 2)
	s.Require().NoError(err)
	s.Equal(2, moved)
	s.Equal(uint32(50), next)

	next, moved, err = s.storage.Reassign(ctx, "dz", "fr", next, 2)
	s.Require().NoError(err)
	s.Equal(1, moved)
	s.Equal(uint32(0), next)

	s.Equal(0, s.storage.Held("dz"))
}

func (s *TileStorageContractSuite) TestReassignOfACountryWithNoTilesDoesNothing() {
	ctx := context.Background()
	s.Require().NoError(s.storage.Set(ctx, 1, "fr"))

	next, moved, err := s.storage.Reassign(ctx, "dz", "fr", 0, 10)
	s.Require().NoError(err)
	s.Equal(uint32(0), next)
	s.Zero(moved)

	next, moved, err = s.storage.Reassign(ctx, "fr", "fr", 0, 10)
	s.Require().NoError(err)
	s.Equal(uint32(0), next)
	s.Zero(moved, "a country is not reassigned to itself")
}

func (s *TileStorageContractSuite) TestReassignRefusesANonPositiveLimit() {
	_, _, err := s.storage.Reassign(context.Background(), "dz", "fr", 0, 0)
	s.Error(err)
}

func (s *TileStorageContractSuite) TestRestoreGivesBackOnlyTheTilesStillHoldingFrom() {
	ctx := context.Background()
	for tile, country := range map[uint32]string{3: "ps", 4: "de", 5: "ps"} {
		s.Require().NoError(s.storage.Set(ctx, tile, country))
	}

	restored, err := s.storage.Restore(ctx, []Restoration{
		{Tile: 3, From: "ps", To: "il"},
		{Tile: 4, From: "ps", To: "il"},
		{Tile: 5, From: "ps", To: ""},
		{Tile: contractMaxIndex + 1, From: "ps", To: "il"},
	})
	s.Require().NoError(err)

	s.Equal(2, restored)
	s.Equal(map[uint32]string{3: "il", 4: "de"}, s.owners(0, 10))
	s.Equal(0, s.storage.Held("ps"))
	s.Equal(1, s.storage.Held("il"))
}

func (s *TileStorageContractSuite) TestRestorePublishesOneOrdinaryUpdatePerTile() {
	s.Require().NoError(s.storage.Set(context.Background(), 7, "ps"))
	listener, ctx := s.subscribe(2 * time.Second)

	_, err := s.storage.Restore(ctx, []Restoration{{Tile: 7, From: "ps", To: "il"}})
	s.Require().NoError(err)

	s.Equal(&TileUpdate{Tile: 7, Value: "il", Previous: "ps"}, s.next(ctx, listener).Update)
}

func (s *TileStorageContractSuite) shield(tile uint32, country string, shields int) {
	for range shields {
		s.Require().NoError(s.storage.Shield(context.Background(), tile, country, 10))
	}
}

func (s *TileStorageContractSuite) TestShieldPublishesTheShieldsStanding() {
	s.Require().NoError(s.storage.Set(context.Background(), 7, "fr"))
	listener, ctx := s.subscribe(2 * time.Second)

	s.Require().NoError(s.storage.Shield(ctx, 7, "fr", 10))
	s.Require().NoError(s.storage.Shield(ctx, 7, "fr", 10))

	s.Equal(&TileUpdate{Tile: 7, Value: "fr", Previous: "fr", Shields: 1}, s.next(ctx, listener).Update)
	s.Equal(&TileUpdate{Tile: 7, Value: "fr", Previous: "fr", Shields: 2}, s.next(ctx, listener).Update)
	s.Equal(2, s.storage.Shields(7))
}

func (s *TileStorageContractSuite) TestShieldRefusesATileOfAnotherFlagAndAFullOne() {
	ctx := context.Background()
	s.Require().NoError(s.storage.Set(ctx, 7, "fr"))

	s.Require().ErrorIs(s.storage.Shield(ctx, 7, "de", 10), ErrNotYourTile)
	s.Require().ErrorIs(s.storage.Shield(ctx, 8, "fr", 10), ErrNotYourTile, "an empty tile wears no flag")
	s.Require().NoError(s.storage.Shield(ctx, 7, "fr", 1))
	s.Require().ErrorIs(s.storage.Shield(ctx, 7, "fr", 1), ErrTileFull)
	s.Require().ErrorIs(s.storage.Shield(ctx, contractMaxIndex+1, "fr", 1), ErrTileOutOfRange)

	s.Equal(1, s.storage.Shields(7))
}

func (s *TileStorageContractSuite) TestAStrikeTakesOneShieldAndPublishesWhatIsLeft() {
	s.Require().NoError(s.storage.Set(context.Background(), 7, "fr"))
	s.shield(7, "fr", 1)
	listener, ctx := s.subscribe(2 * time.Second)

	s.True(s.storage.Strike(ctx, 7, "fr"))
	s.False(s.storage.Strike(ctx, 7, "fr"), "no shield is left to strike")

	s.Equal(&TileUpdate{Tile: 7, Value: "fr", Previous: "fr"}, s.next(ctx, listener).Update)
	s.Empty(listener)
}

func (s *TileStorageContractSuite) TestAStrikeAgainstAnOwnerTheTileNoLongerHasTakesNothing() {
	ctx := context.Background()
	s.Require().NoError(s.storage.Set(ctx, 7, "fr"))
	s.shield(7, "fr", 2)

	s.False(s.storage.Strike(ctx, 7, "de"))
	s.Equal(2, s.storage.Shields(7))
}

func (s *TileStorageContractSuite) TestEveryChangeOfOwnerSendsTheShieldsHome() {
	ctx := context.Background()
	for tile := uint32(1); tile <= 4; tile++ {
		s.Require().NoError(s.storage.Set(ctx, tile, "fr"))
		s.shield(tile, "fr", 3)
	}

	s.Require().NoError(s.storage.Click(ctx, 1, "de"))
	_, _, err := s.storage.Reassign(ctx, "fr", "it", 2, 1)
	s.Require().NoError(err)
	_, err = s.storage.Restore(ctx, []Restoration{{Tile: 3, From: "fr", To: "es"}})
	s.Require().NoError(err)
	s.Require().NoError(s.storage.Set(ctx, 4, "fr"))

	s.Zero(s.storage.Shields(1))
	s.Zero(s.storage.Shields(2))
	s.Zero(s.storage.Shields(3))
	s.Equal(3, s.storage.Shields(4), "a write that changes no owner keeps them")
}

func (s *TileStorageContractSuite) TestABlastStrikesTheShieldedTilesAndClearsTheRest() {
	ctx := context.Background()
	for tile := uint32(10); tile <= 12; tile++ {
		s.Require().NoError(s.storage.Set(ctx, tile, "fr"))
	}
	s.shield(11, "fr", 2)

	blast, err := s.storage.Clear(ctx, Blast{Cleared: []uint32{10, 11, 12}})
	s.Require().NoError(err)

	s.Equal([]uint32{10, 12}, blast.Cleared)
	s.Equal([]uint32{11}, blast.Struck)
	s.Equal([]int{1}, blast.Left, "with the shields each one kept")
	s.Equal(map[uint32]string{11: "fr"}, s.owners(10, 12))
	s.Equal(1, s.storage.Shields(11))
}

func (s *TileStorageContractSuite) TestAStateBatchCarriesTheShieldedTilesInRange() {
	ctx := context.Background()
	for _, tile := range []uint32{5, 20, 30} {
		s.Require().NoError(s.storage.Set(ctx, tile, "fr"))
	}
	s.shield(5, "fr", 1)
	s.shield(20, "fr", 3)

	batch, err := s.storage.StateBatchDense(10, 40)
	s.Require().NoError(err)

	s.Equal([]TileShields{{Tile: 20, Shields: 3}}, batch.Shields)
}

func (s *TileStorageContractSuite) own(flag string, tiles ...uint32) {
	for _, tile := range tiles {
		s.Require().NoError(s.storage.Set(context.Background(), tile, flag))
	}
}

func (s *TileStorageContractSuite) shieldsOn(tiles ...uint32) []int {
	shields := make([]int, len(tiles))
	for i, tile := range tiles {
		shields[i] = s.storage.Shields(tile)
	}
	return shields
}

func (s *TileStorageContractSuite) TestAWholeLandmassFortifiedGivesEachOfItsTilesAShield() {
	s.own("fr", contractIsland...)
	listener, ctx := s.subscribe(time.Second)

	fortification, err := s.storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)

	s.Equal(Fortification{Landmass: 1, Ground: "l1", Country: "fr", Tile: 90_003, Tiles: 3, Raised: []TileShields{
		{Tile: 90_001, Shields: 1}, {Tile: 90_002, Shields: 1}, {Tile: 90_003, Shields: 1},
	}}, fortification)
	s.Equal([]int{1, 1, 1}, s.shieldsOn(contractIsland...))
	s.Equal(0, s.storage.Shields(90_004))
	change := s.next(ctx, listener)
	s.Require().NotNil(change.Fortification)
	s.Equal(fortification, *change.Fortification)
}

func (s *TileStorageContractSuite) TestTheFortressesSayWhoFortifiedEachLandmassLast() {
	ctx := context.Background()
	s.Empty(s.storage.Fortresses())

	s.own("fr", contractIsland...)
	_, err := s.storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)
	s.own("dz", contractIslet...)
	_, _, err = s.storage.Reassign(ctx, "dz", "de", 0, 100)
	s.Require().NoError(err)

	s.Equal(map[LandmassID]string{1: "fr", 2: "de"}, s.storage.Fortresses())
}

func (s *TileStorageContractSuite) TestALandmassNotWholeIsNotFortified() {
	s.own("fr", 90_001, 90_002)
	s.own("de", 90_003)

	_, err := s.storage.Fortify(context.Background(), 90_002, "fr", 10)

	s.Require().ErrorIs(err, ErrNotWhole)
	s.Equal([]int{0, 0, 0}, s.shieldsOn(contractIsland...))
}

func (s *TileStorageContractSuite) TestATileOnNoLandmassIsNeverFortified() {
	s.own("fr", 5)

	_, err := s.storage.Fortify(context.Background(), 5, "fr", 10)

	s.Require().ErrorIs(err, ErrNotWhole)
	s.Equal(0, s.storage.Shields(5))
}

func (s *TileStorageContractSuite) TestAFlagCannotFortifyTheSameLandmassTwiceInARow() {
	ctx := context.Background()
	s.own("fr", contractIsland...)
	_, err := s.storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)

	s.own("de", 90_001)
	s.own("fr", 90_001)
	_, err = s.storage.Fortify(ctx, 90_001, "fr", 10)

	s.Require().ErrorIs(err, ErrFortifiedAlready)
	s.Equal([]int{0, 1, 1}, s.shieldsOn(contractIsland...))
}

func (s *TileStorageContractSuite) TestAnotherFlagFortifyingALandmassLetsTheFirstFortifyItAgain() {
	ctx := context.Background()
	s.own("fr", contractIsland...)
	_, err := s.storage.Fortify(ctx, 90_003, "fr", 10)
	s.Require().NoError(err)

	s.own("de", contractIsland...)
	_, err = s.storage.Fortify(ctx, 90_003, "de", 10)
	s.Require().NoError(err)
	s.own("fr", contractIsland...)
	_, err = s.storage.Fortify(ctx, 90_002, "fr", 10)

	s.Require().NoError(err)
	s.Equal([]int{1, 1, 1}, s.shieldsOn(contractIsland...))
}

func (s *TileStorageContractSuite) TestAFortifyStopsAtTheMostATileHolds() {
	ctx := context.Background()
	s.own("fr", contractIsland...)
	s.Require().NoError(s.storage.Shield(ctx, 90_001, "fr", 2))
	s.Require().NoError(s.storage.Shield(ctx, 90_001, "fr", 2))

	fortification, err := s.storage.Fortify(ctx, 90_002, "fr", 2)

	s.Require().NoError(err)
	s.Equal([]int{2, 1, 1}, s.shieldsOn(contractIsland...))
	s.Equal([]TileShields{{Tile: 90_002, Shields: 1}, {Tile: 90_003, Shields: 1}}, fortification.Raised)
}

func (s *TileStorageContractSuite) TestABlastBreaksAWholeLandmass() {
	ctx := context.Background()
	s.own("fr", contractIsland...)
	_, err := s.storage.Clear(ctx, Blast{Tile: 90_001, CountryID: "de", Cleared: []uint32{90_001}})
	s.Require().NoError(err)

	_, err = s.storage.Fortify(ctx, 90_002, "fr", 10)

	s.Require().ErrorIs(err, ErrNotWhole)
}

func (s *TileStorageContractSuite) TestAReassignThatMakesALandmassWholeLocksItToItsNewHolder() {
	ctx := context.Background()
	s.own("dz", contractIsland...)
	_, _, err := s.storage.Reassign(ctx, "dz", "fr", 0, 100)
	s.Require().NoError(err)

	_, err = s.storage.Fortify(ctx, 90_001, "fr", 10)

	s.Require().ErrorIs(err, ErrFortifiedAlready)
	s.Equal([]int{0, 0, 0}, s.shieldsOn(contractIsland...))
}

func (s *TileStorageContractSuite) TestARestoreThatMakesALandmassWholeLocksItToItsNewHolder() {
	ctx := context.Background()
	s.own("fr", 90_001, 90_002)
	s.own("ps", 90_003)
	_, err := s.storage.Restore(ctx, []Restoration{{Tile: 90_003, From: "ps", To: "fr"}})
	s.Require().NoError(err)

	_, err = s.storage.Fortify(ctx, 90_003, "fr", 10)

	s.Require().ErrorIs(err, ErrFortifiedAlready)
}
