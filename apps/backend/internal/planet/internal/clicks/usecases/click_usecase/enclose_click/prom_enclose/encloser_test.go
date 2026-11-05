package prom_enclose_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click/prom_enclose"
)

type encloser struct {
	enclosed []clicks.Enclosure
	err      error
}

func (e *encloser) Enclose(_ context.Context, enclosure clicks.Enclosure, _ []uint32) error {
	e.enclosed = append(e.enclosed, enclosure)
	return e.err
}

func TestEveryShapeIsCountedWithTheTilesItTookAndStillEnclosed(t *testing.T) {
	registry := prometheus.NewRegistry()
	inner := &encloser{}

	counted := prom_enclose.New(inner, registry)

	require.NoError(t, counted.Enclose(t.Context(), clicks.Enclosure{ClosingTile: 1}, []uint32{1, 2, 3}))
	require.NoError(t, counted.Enclose(t.Context(), clicks.Enclosure{ClosingTile: 4}, []uint32{4}))

	assert.Len(t, inner.enclosed, 2)
	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP bonus_enclosed_tiles_total Tiles taken inside shapes closed with the enclose bonus
# TYPE bonus_enclosed_tiles_total counter
bonus_enclosed_tiles_total 4
# HELP bonus_enclosures_total Clicks that closed shapes with the enclose bonus
# TYPE bonus_enclosures_total counter
bonus_enclosures_total 2
`)))
}

func TestAFailedEnclosureIsNotCounted(t *testing.T) {
	registry := prometheus.NewRegistry()
	failed := errors.New("tile out of range")

	err := prom_enclose.New(&encloser{err: failed}, registry).Enclose(t.Context(), clicks.Enclosure{}, []uint32{1})

	require.ErrorIs(t, err, failed)
	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP bonus_enclosed_tiles_total Tiles taken inside shapes closed with the enclose bonus
# TYPE bonus_enclosed_tiles_total counter
bonus_enclosed_tiles_total 0
# HELP bonus_enclosures_total Clicks that closed shapes with the enclose bonus
# TYPE bonus_enclosures_total counter
bonus_enclosures_total 0
`)))
}
