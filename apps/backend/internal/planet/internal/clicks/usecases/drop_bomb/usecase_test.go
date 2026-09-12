package drop_bomb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubBombs struct {
	held  bool
	taken string
}

func (s *stubBombs) Take(scope string) bool {
	s.taken = scope
	held := s.held
	s.held = false
	return held
}

type stubSchedule struct{ dropped []string }

func (s *stubSchedule) Dropped(scope string) { s.dropped = append(s.dropped, scope) }

type stubMap struct {
	nearest uint32
	arc     float64
	centre  clicks.Vec3
	radius  float64
}

func (s *stubMap) Nearest(clicks.Vec3) (uint32, float64) { return s.nearest, s.arc }

func (s *stubMap) Position(uint32) (clicks.Vec3, bool) { return clicks.Vec3{Y: 1}, true }

func (s *stubMap) Within(centre clicks.Vec3, radius float64) []uint32 {
	s.centre, s.radius = centre, radius
	return []uint32{9, 10, 11}
}

type stubClearer struct{ cleared []clicks.Blast }

func (s *stubClearer) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	s.cleared = append(s.cleared, blast)
	return blast, nil
}

type countries struct{}

func (countries) CheckCountry(country string) bool { return country == "fr" }

var rules = drop_bomb.Rules{Radius: 0.03, Reach: 0.005}

type parts struct {
	bombs    *stubBombs
	schedule *stubSchedule
	geo      *stubMap
	clearer  *stubClearer
}

func setup(held bool, arc float64) (*drop_bomb.UseCase, parts) {
	p := parts{
		bombs:    &stubBombs{held: held},
		schedule: &stubSchedule{},
		geo:      &stubMap{nearest: 10, arc: arc},
		clearer:  &stubClearer{},
	}

	return drop_bomb.New(p.bombs, p.schedule, p.geo, p.clearer, countries{}, rules), p
}

func TestABombOnLandClearsACircleAroundTheTileHit(t *testing.T) {
	useCase, p := setup(true, 0.001)

	blast, err := useCase.Execute(t.Context(), drop_bomb.In{Target: clicks.Vec3{X: 2}, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, uint32(10), blast.Tile)
	assert.Equal(t, []uint32{9, 10, 11}, blast.Cleared)
	assert.Equal(t, clicks.Vec3{Y: 1}, p.geo.centre, "the circle is centred on the tile hit, not the raw aim")
	assert.InDelta(t, 0.03, p.geo.radius, 1e-9)
	assert.InDelta(t, 0.03, blast.Radius, 1e-9)
	assert.Equal(t, clicks.Vec3{Y: 1}, blast.Point, "drawn at the tile's centre")
	assert.Equal(t, []string{cpctx.RateLimitKey(t.Context())}, p.schedule.dropped)
}

func TestABombInTheSeaIsSpentAndClearsNothing(t *testing.T) {
	useCase, p := setup(true, 0.2)

	blast, err := useCase.Execute(t.Context(), drop_bomb.In{Target: clicks.Vec3{X: 2}, CountryID: "fr"})
	require.NoError(t, err)

	assert.Zero(t, blast.Tile)
	assert.Empty(t, blast.Cleared)
	assert.Equal(t, clicks.Vec3{X: 1}, blast.Point, "drawn where it was aimed, on the unit sphere")
	assert.Len(t, p.clearer.cleared, 1, "a splash is still announced")
	assert.False(t, p.bombs.held, "the bomb is gone")
}

func TestNoBombNoBlast(t *testing.T) {
	useCase, p := setup(false, 0.001)

	_, err := useCase.Execute(t.Context(), drop_bomb.In{Target: clicks.Vec3{X: 1}, CountryID: "fr"})

	require.ErrorIs(t, err, drop_bomb.ErrNoBomb)
	assert.Empty(t, p.clearer.cleared)
	assert.Empty(t, p.schedule.dropped)
}

func TestAMalformedDropDoesNotCostTheBomb(t *testing.T) {
	useCase, p := setup(true, 0.001)

	_, err := useCase.Execute(t.Context(), drop_bomb.In{Target: clicks.Vec3{X: 1}, CountryID: "xx"})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)

	p.geo.nearest = 0
	_, err = useCase.Execute(t.Context(), drop_bomb.In{CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrTileOutOfRange)

	assert.True(t, p.bombs.held)
}
