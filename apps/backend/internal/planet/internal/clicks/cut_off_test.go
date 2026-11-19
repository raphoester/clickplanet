package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The tiles next to each other, and the bridges, both ways.
type links struct {
	next    map[uint32][]uint32
	bridges map[uint32][]uint32
}

func linksOf(pairs ...[2]uint32) links {
	l := links{next: map[uint32][]uint32{}, bridges: map[uint32][]uint32{}}
	for _, pair := range pairs {
		l.next[pair[0]] = append(l.next[pair[0]], pair[1])
		l.next[pair[1]] = append(l.next[pair[1]], pair[0])
	}
	return l
}

func (l links) bridge(a, b uint32) links {
	l.bridges[a] = append(l.bridges[a], b)
	l.bridges[b] = append(l.bridges[b], a)
	return l
}

func (l links) Neighbours(id uint32) []uint32 { return l.next[id] }

func (l links) Across(tile uint32) []uint32 { return l.bridges[tile] }

// France's capital is tile 1.
func cutOffPlay(rules clicks.CapitalRules) clicks.CapitalPlay {
	return clicks.NewCapitalPlay(
		func() *clicks.Capitals { return clicks.NewCapitals(map[string]uint32{"fr": 1}) },
		func() clicks.CapitalRules { return rules },
	)
}

// heldAfter is the land France holds once the lost tiles left it.
func heldAfter(land []uint32, lost ...uint32) func(uint32) bool {
	held := cpcolls.NewSet(land...)
	held.Delete(lost...)
	return held.Contains
}

var cuts = clicks.CapitalRules{CutOff: true}

// 1 - 2 - 3 - 4 - 5
var corridor = linksOf([2]uint32{1, 2}, [2]uint32{2, 3}, [2]uint32{3, 4}, [2]uint32{4, 5})

func TestATakeInACorridorStrandsTheLandPastIt(t *testing.T) {
	cut, err := cutOffPlay(cuts).Stranded("fr", []uint32{3}, corridor, heldAfter([]uint32{1, 2, 3, 4, 5}, 3))

	require.NoError(t, err)
	assert.Equal(t, []uint32{4, 5}, cut)
}

func TestATakeAtTheTipOfTheLandStrandsNothing(t *testing.T) {
	_, err := cutOffPlay(cuts).Stranded("fr", []uint32{5}, corridor, heldAfter([]uint32{1, 2, 3, 4, 5}, 5))

	require.ErrorIs(t, err, clicks.ErrNoCutOff)
}

func TestATakeInALoopStrandsNothing(t *testing.T) {
	// 1 - 2 - 3 - 4 - 1: the land around 2 still holds to the capital the other way.
	loop := linksOf([2]uint32{1, 2}, [2]uint32{2, 3}, [2]uint32{3, 4}, [2]uint32{4, 1})

	_, err := cutOffPlay(cuts).Stranded("fr", []uint32{2}, loop, heldAfter([]uint32{1, 2, 3, 4}, 2))

	require.ErrorIs(t, err, clicks.ErrNoCutOff)
}

func TestLandHoldsToItsCapitalAcrossABridgeWhoseBothEndsItHolds(t *testing.T) {
	// 1 - 2 ~ 10 - 11, and 2 - 3.
	coast := linksOf([2]uint32{1, 2}, [2]uint32{2, 3}, [2]uint32{10, 11}).bridge(2, 10)
	land := []uint32{1, 2, 3, 10, 11}

	_, err := cutOffPlay(cuts).Stranded("fr", []uint32{3}, coast, heldAfter(land, 3))
	require.ErrorIs(t, err, clicks.ErrNoCutOff, "the island holds across the bridge")

	cut, err := cutOffPlay(cuts).Stranded("fr", []uint32{2}, coast, heldAfter(land, 2))
	require.NoError(t, err)
	assert.Equal(t, []uint32{3, 10, 11}, cut, "the bridge's end taken strands the island")
}

func TestTilesLostTogetherAreReadTogether(t *testing.T) {
	// 1 - 2 - 3 - 4 - 5: each of 3 and 4 alone has one tile of the land on its edge.
	cut, err := cutOffPlay(cuts).Stranded("fr", []uint32{3, 4}, corridor, heldAfter([]uint32{1, 2, 3, 4, 5}, 3, 4))

	require.NoError(t, err)
	assert.Equal(t, []uint32{5}, cut)
}

func TestAFlagThatLostItsCapitalLosesTheLandNextToIt(t *testing.T) {
	cut, err := cutOffPlay(cuts).Stranded("fr", []uint32{1}, corridor, heldAfter([]uint32{1, 2, 3, 4, 5}, 1))

	require.NoError(t, err)
	assert.Equal(t, []uint32{2, 3, 4, 5}, cut)
}

func TestLandCutOffBeforeStays(t *testing.T) {
	// 7 holds to nothing of France's, and nothing lost touches it.
	land := linksOf([2]uint32{1, 2}, [2]uint32{2, 3}, [2]uint32{6, 7})

	_, err := cutOffPlay(cuts).Stranded("fr", []uint32{3}, land, heldAfter([]uint32{1, 2, 3, 7}, 3))

	require.ErrorIs(t, err, clicks.ErrNoCutOff)
}

func TestNothingIsStrandedWhileTheCutOffDoesNotPlay(t *testing.T) {
	_, err := cutOffPlay(clicks.CapitalRules{KnockOut: true}).
		Stranded("fr", []uint32{3}, corridor, heldAfter([]uint32{1, 2, 3, 4, 5}, 3))

	require.ErrorIs(t, err, clicks.ErrNoCutOff)
}

func TestAFlagWithNoCapitalHasNoLandToCutFromIt(t *testing.T) {
	_, err := cutOffPlay(cuts).Stranded("gb-eng", []uint32{3}, corridor, heldAfter([]uint32{1, 2, 3, 4, 5}, 3))

	require.ErrorIs(t, err, clicks.ErrNoCutOff)
}

func TestTheCapitalsPartReadsTheCutOff(t *testing.T) {
	on := true

	rules := clicks.CapitalsPart.Read(&planetv1.Rules{Capitals: &planetv1.CapitalRules{CutOff: &on}})

	assert.Equal(t, clicks.CapitalRules{CutOff: true}, rules)
}
