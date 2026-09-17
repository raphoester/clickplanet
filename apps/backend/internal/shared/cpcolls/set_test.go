package cpcolls_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func TestASetHoldsEachItemOnce(t *testing.T) {
	set := cpcolls.NewSet("a", "b", "a")

	set.Add("b", "c")

	assert.Equal(t, 3, set.Len())
	assert.ElementsMatch(t, []string{"a", "b", "c"}, collect(set))
}

func TestASetForgetsWhatIsDeleted(t *testing.T) {
	set := cpcolls.NewSet(1, 2, 3)

	set.Delete(2, 4)

	assert.True(t, set.Contains(1))
	assert.False(t, set.Contains(2))
	assert.Equal(t, 2, set.Len())
}

func TestAddSetUnitesTwoSets(t *testing.T) {
	set := cpcolls.NewSet(1, 2)

	set.AddSet(cpcolls.NewSet(2, 3))

	assert.ElementsMatch(t, []int{1, 2, 3}, collect(set))
}

func TestClearEmptiesASetThatStaysUsable(t *testing.T) {
	set := cpcolls.NewSetWithCapacity[string](2)
	set.Add("a")

	set.Clear()
	set.Add("b")

	assert.False(t, set.Contains("a"))
	assert.True(t, set.Contains("b"))
	assert.Equal(t, 1, set.Len())
}

func TestANilSetReadsAsEmpty(t *testing.T) {
	var set *cpcolls.Set[string]

	assert.False(t, set.Contains("a"))
	assert.Zero(t, set.Len())
	assert.Empty(t, collect(set))
}

func TestForEachVisitsEveryItemOnce(t *testing.T) {
	visits := map[int]int{}

	cpcolls.NewSet(1, 2, 3).ForEach(func(item int) { visits[item]++ })

	assert.Equal(t, map[int]int{1: 1, 2: 1, 3: 1}, visits)
}

func collect[T comparable](set *cpcolls.Set[T]) []T {
	var items []T
	set.ForEach(func(item T) { items = append(items, item) })
	return items
}
