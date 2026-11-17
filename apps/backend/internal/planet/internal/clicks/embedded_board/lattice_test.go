package embedded_board

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func TestTheWalkVisitsEveryPointOnce(t *testing.T) {
	for _, cols := range []int{2, 5, 13} {
		seen := cpcolls.NewSet[canonicalID]()
		walkLattice(cols, func(_, _, _ int, id canonicalID) {
			assert.False(t, seen.Contains(id), "cols %d: point %d twice", cols, id)
			seen.Add(id)
		})

		assert.Equal(t, latticeSize(cols), seen.Len(), "cols %d", cols)
		for id := range canonicalID(latticeSize(cols)) {
			require.True(t, seen.Contains(id), "cols %d: point %d never", cols, id)
		}
	}
}

func TestEveryPointIsOnTheUnitSphere(t *testing.T) {
	walkLattice(7, func(face, i, j int, id canonicalID) {
		p := latticePosition(7, face, i, j)
		length := math.Sqrt(float64(p[0])*float64(p[0]) + float64(p[1])*float64(p[1]) + float64(p[2])*float64(p[2]))
		assert.InDelta(t, 1, length, 1e-6, "point %d", id)
	})
}

func TestEveryPointTouches6AndTheCorners5(t *testing.T) {
	cols := 6
	tileOf := make([]uint32, latticeSize(cols))
	for id := range tileOf {
		tileOf[id] = uint32(id) + 1 //nolint:gosec // a lattice of 362 points.
	}

	geography, err := clicks.NewGeography(make([]float32, len(tileOf)*3), latticeEdges(cols, tileOf))
	require.NoError(t, err)

	assert.Equal(t, [clicks.MaxDegree + 1]uint32{5: 12, 6: uint32(latticeSize(cols)) - 12}, geography.Stats().Degrees) //nolint:gosec // 362 points.
	for corner := range uint32(cornerCount) {
		assert.Len(t, geography.Neighbours(corner+1), 5, "corner %d is a canonical id below 12", corner)
	}
}

func TestEdgesJoinOnlyTiles(t *testing.T) {
	cols := 4
	tileOf := make([]uint32, latticeSize(cols))
	tileOf[20], tileOf[21] = 1, 2

	for _, edge := range latticeEdges(cols, tileOf) {
		assert.NotZero(t, edge.From)
		assert.NotZero(t, edge.To)
	}
}

func runsFile(runs ...[2]uint64) []byte {
	blob := make([]byte, runsHeader+len(runs)*runBytes)
	copy(blob, runsMagic)
	binary.LittleEndian.PutUint32(blob[4:], runsVersion)
	binary.LittleEndian.PutUint32(blob[8:], uint32(len(runs))) //nolint:gosec // a few runs.
	for k, run := range runs {
		binary.LittleEndian.PutUint64(blob[runsHeader+k*runBytes:], run[0])
		binary.LittleEndian.PutUint64(blob[runsHeader+k*runBytes+8:], run[1])
	}
	return blob
}

func TestRunsAreDecoded(t *testing.T) {
	runs, err := decodeRuns(runsFile([2]uint64{0, 3}, [2]uint64{7, 7}), 10)
	require.NoError(t, err)
	assert.Equal(t, []run{{first: 0, last: 3}, {first: 7, last: 7}}, runs)
}

func TestRunsThatAreNotTheLatticesAreRefused(t *testing.T) {
	for name, blob := range map[string][]byte{
		"truncated":         runsFile([2]uint64{0, 1})[:20],
		"past the lattice":  runsFile([2]uint64{0, 10}),
		"backwards":         runsFile([2]uint64{4, 3}),
		"out of order":      runsFile([2]uint64{5, 6}, [2]uint64{1, 2}),
		"overlapping":       runsFile([2]uint64{1, 4}, [2]uint64{4, 6}),
		"another magic":     append([]byte("CPLM"), runsFile()[4:]...),
		"another version":   append(runsFile()[:4], 2, 0, 0, 0, 0, 0, 0, 0),
		"shorter than head": []byte("CPRN"),
	} {
		_, err := decodeRuns(blob, 10)
		assert.Error(t, err, name)
	}
}
