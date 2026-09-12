package prom_enclose_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/enclose_click/prom_enclose"
)

type recorder struct{ published []bonus.Enclosed }

func (r *recorder) PublishEnclosed(_ string, enclosed bonus.Enclosed) {
	r.published = append(r.published, enclosed)
}

func TestEveryShapeIsCountedWithTheTilesItTookAndStillPublished(t *testing.T) {
	registry := prometheus.NewRegistry()
	inner := &recorder{}

	publisher, err := prom_enclose.New(inner, registry)
	require.NoError(t, err)

	publisher.PublishEnclosed("scope-a", bonus.Enclosed{Filled: []uint32{1, 2, 3}})
	publisher.PublishEnclosed("scope-a", bonus.Enclosed{Filled: []uint32{4}})

	assert.Len(t, inner.published, 2)
	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP bonus_enclosed_tiles_total Tiles taken inside shapes closed with the enclose bonus
# TYPE bonus_enclosed_tiles_total counter
bonus_enclosed_tiles_total 4
# HELP bonus_enclosures_total Shapes closed with the enclose bonus
# TYPE bonus_enclosures_total counter
bonus_enclosures_total 2
`)))
}
