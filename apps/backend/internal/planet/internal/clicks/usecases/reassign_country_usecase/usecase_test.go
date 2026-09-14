package reassign_country_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/pacing"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
)

type stubMap struct {
	tiles  []string
	limits []int
	starts []uint32
	err    error
}

func (m *stubMap) Held(country string) int {
	n := 0
	for _, owner := range m.tiles {
		if owner == country {
			n++
		}
	}
	return n
}

func (m *stubMap) Reassign(_ context.Context, from, to string, start uint32, limit int) (uint32, int, error) {
	m.starts = append(m.starts, start)
	m.limits = append(m.limits, limit)
	if m.err != nil {
		return 0, 0, m.err
	}

	moved := 0
	for tile := int(start); tile < len(m.tiles); tile++ {
		if m.tiles[tile] != from {
			continue
		}
		if moved == limit {
			return uint32(tile), moved, nil //nolint:gosec // test data is tiny.
		}
		m.tiles[tile] = to
		moved++
	}
	return 0, moved, nil
}

type countries struct{}

func (countries) CheckCountry(country string) bool {
	return country == "dz" || country == "fr" || country == "bg"
}

func newMap() *stubMap {
	return &stubMap{tiles: []string{"", "dz", "bg", "dz", "dz", "fr", "dz", "dz"}}
}

func TestItMovesEveryTileInBatches(t *testing.T) {
	tiles := newMap()
	useCase := reassign_country_usecase.New(tiles, countries{}, pacing.Pacing{Batch: 2})

	out, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: "fr"})
	require.NoError(t, err)

	assert.Equal(t, reassign_country_usecase.Out{FromBefore: 5, ToBefore: 1, Moved: 5, FromAfter: 0, ToAfter: 6}, out)
	assert.Equal(t, []uint32{0, 4, 7}, tiles.starts, "each batch resumes where the last stopped")
	assert.Equal(t, []int{2, 2, 2}, tiles.limits)
	assert.Equal(t, "bg", tiles.tiles[2], "a tile somebody else holds is left alone")
}

func TestADryRunCountsAndMovesNothing(t *testing.T) {
	tiles := newMap()
	useCase := reassign_country_usecase.New(tiles, countries{}, pacing.Pacing{Batch: 2})

	out, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: "fr", DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, reassign_country_usecase.Out{FromBefore: 5, ToBefore: 1, FromAfter: 5, ToAfter: 1}, out)
	assert.Empty(t, tiles.starts)
}

func TestItRefusesAnUnknownCountryOnEitherSide(t *testing.T) {
	useCase := reassign_country_usecase.New(newMap(), countries{}, pacing.Pacing{Batch: 2})

	_, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "xx", To: "fr"})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)

	_, err = useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: ""})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry, "clearing a country is a bomb's job, not this one")
}

func TestItRefusesACountryToItself(t *testing.T) {
	useCase := reassign_country_usecase.New(newMap(), countries{}, pacing.Pacing{Batch: 2})

	_, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "fr", To: "fr"})
	require.ErrorIs(t, err, reassign_country_usecase.ErrSameCountry)
}

func TestItStopsWhenTheContextEndsAndSaysHowFarItGot(t *testing.T) {
	tiles := newMap()
	useCase := reassign_country_usecase.New(tiles, countries{}, pacing.Pacing{Batch: 2, Pause: time.Hour})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := useCase.Execute(ctx, reassign_country_usecase.In{From: "dz", To: "fr"})
	require.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, 2, out.Moved, "the first batch went before the pause noticed")
	assert.Equal(t, 3, out.FromAfter)
	assert.Equal(t, 3, out.ToAfter)
}

func TestAStorageErrorIsReturned(t *testing.T) {
	tiles := newMap()
	tiles.err = errors.New("code table full")
	useCase := reassign_country_usecase.New(tiles, countries{}, pacing.Pacing{Batch: 2})

	_, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: "fr"})
	require.ErrorIs(t, err, tiles.err)
}
