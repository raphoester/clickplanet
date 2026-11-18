package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type positions map[uint32]clicks.Vec3

func (p positions) Position(tile uint32) (clicks.Vec3, bool) {
	at, ok := p[tile]
	return at, ok
}

// Old tile 1 has children 1 to 4, old 2 has 6, old 3 none; new tile 5 comes from nowhere.
func carryover(t *testing.T) clicks.Carryover {
	t.Helper()
	carry, err := clicks.NewCarryover("earth-1", "earth-2", 3, []uint32{1, 1, 1, 1, 0, 2})
	require.NoError(t, err)
	return carry
}

func TestACarryOverNamesEachNewTilesParentAndEachOldTilesChildrenInIdOrder(t *testing.T) {
	carry := carryover(t)

	assert.Equal(t, uint32(6), carry.Tiles())
	assert.Equal(t, uint32(3), carry.OldTiles())
	assert.Equal(t, uint32(1), carry.Parent(3))
	assert.Equal(t, uint32(0), carry.Parent(5), "a new coast comes from nowhere")
	assert.Equal(t, uint32(0), carry.Parent(7), "past the new board")
	assert.Equal(t, []uint32{1, 2, 3, 4}, carry.Children(1))
	assert.Equal(t, []uint32{6}, carry.Children(2))
	assert.Empty(t, carry.Children(3), "a coast the new board draws as sea has no child")
	assert.Empty(t, carry.Children(4))
}

func TestAParentsShieldsAreSplitEvenlyInIdOrderTheRemainderToTheFirstChildren(t *testing.T) {
	carry := carryover(t)

	shares := make([]int, 0, 4)
	for _, child := range carry.Children(1) {
		shares = append(shares, carry.Share(child, 10))
	}
	assert.Equal(t, []int{3, 3, 2, 2}, shares, "10 over four, the total kept")
	assert.Equal(t, 1, carry.Share(1, 1))
	assert.Equal(t, 0, carry.Share(2, 1))
	assert.Equal(t, 7, carry.Share(6, 7), "an only child keeps them all")
	assert.Equal(t, 0, carry.Share(5, 7), "a tile with no parent gets none")
	assert.Equal(t, 0, carry.Share(1, 0))
}

func TestAMineGoesToTheChildNearestItsTile(t *testing.T) {
	carry := carryover(t)
	old := positions{1: {X: 1}, 2: {Y: 1}, 3: {Z: 1}}
	fresh := positions{1: {X: 0.9, Y: 0.1}, 2: {X: 0.99, Z: 0.01}, 3: {X: 0.99, Y: 0.01}, 4: {X: 0.5}, 6: {Y: 0.9}}

	heir, ok := carry.Heir(1, old, fresh)
	require.True(t, ok)
	assert.Equal(t, uint32(2), heir, "the first of two as near, in id order")

	heir, ok = carry.Heir(2, old, fresh)
	require.True(t, ok)
	assert.Equal(t, uint32(6), heir)

	_, ok = carry.Heir(3, old, fresh)
	assert.False(t, ok, "a mine on a tile with no child is lost")
}

func TestACarryOverRefusesAParentPastTheOldBoard(t *testing.T) {
	_, err := clicks.NewCarryover("earth-1", "earth-2", 3, []uint32{1, 4})
	require.ErrorIs(t, err, clicks.ErrCarryover)

	_, err = clicks.NewCarryover("earth-1", "earth-2", 3, nil)
	require.ErrorIs(t, err, clicks.ErrCarryover)
}

func TestEachVisitsEveryCarriedTileWithItsParentInIdOrder(t *testing.T) {
	var visited [][2]uint32
	carryover(t).Each(func(child, parent uint32) { visited = append(visited, [2]uint32{child, parent}) })

	assert.Equal(t, [][2]uint32{{1, 1}, {2, 1}, {3, 1}, {4, 1}, {6, 2}}, visited, "new tile 5 comes from nowhere")
}
