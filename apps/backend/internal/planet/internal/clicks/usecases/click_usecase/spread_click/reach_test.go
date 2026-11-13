package spread_click_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/reach_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
)

type line map[uint32][]uint32

func (l line) Neighbours(id uint32) []uint32 { return l[id] }

var row = line{1: {2}, 2: {1, 3}, 3: {2, 4}, 4: {3, 5}, 5: {4}}

type field struct{ owners map[uint32]string }

func (f *field) HoldsAny(flag string, tiles []uint32) bool {
	for _, tile := range tiles {
		if f.owners[tile] == flag {
			return true
		}
	}
	return false
}

func (f *field) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
	f.owners[in.TileID] = in.CountryID
	return click_usecase.Out{Outcome: clicks.Taken}, nil
}

func (f *field) Spread(_ context.Context, _ uint32, flag string, tiles []uint32) error {
	for _, tile := range tiles {
		f.owners[tile] = flag
	}
	return nil
}

type noBridges struct{}

func (noBridges) Across(uint32) []uint32 { return nil }

func reachingChain(t *testing.T, spreadClicks int, rules func() charges.Rules) (click_usecase.IUseCase, *field, *charges.MemoryStore) {
	t.Helper()

	land := &field{owners: map[uint32]string{1: "fr"}}
	hands := charges.NewMemoryStore()
	hands.Keep(caller, charges.Held{SpreadClicks: spreadClicks})
	reach := clicks.NewReach(func() clicks.Reaching { return true }, row, noBridges{}, land)
	spreads := spread_click.New(land, charges.NewCharges(hands, rules), reach, land, &recordingPublisher{})

	return reach_click.New(spreads, reach), land, hands
}

func TestWithTheReachASpreadingClickMustTouchTheLandLikeAnyClick(t *testing.T) {
	chain, land, hands := reachingChain(t, 1, spreading(2, false))

	_, err := chain.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr", Spread: true})

	require.ErrorIs(t, err, clicks.ErrOutOfReach, "two tiles out, even though its spread would touch the land")
	assert.Equal(t, map[uint32]string{1: "fr"}, land.owners)
	assert.Equal(t, 1, hands.Kept(caller).SpreadClicks)
}

func TestWithTheReachASpreadingClickNextToTheLandTakesItsWholeDisc(t *testing.T) {
	chain, land, hands := reachingChain(t, 1, spreading(2, false))

	_, err := chain.Execute(played(t), click_usecase.In{TileID: 2, CountryID: "fr", Spread: true})

	require.NoError(t, err)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr"}, land.owners)
	assert.Zero(t, hands.Kept(caller).SpreadClicks)
}

func TestWithTheReachAnUnlimitedSpreadNextToTheLandTakesItsWholeDiscForNothing(t *testing.T) {
	chain, land, hands := reachingChain(t, 3, spreading(2, true))

	_, err := chain.Execute(played(t), click_usecase.In{TileID: 2, CountryID: "fr", Spread: true})

	require.NoError(t, err)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr"}, land.owners)
	assert.Equal(t, 3, hands.Kept(caller).SpreadClicks)
}
