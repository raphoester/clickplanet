package reach_click_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/reach_click"
)

type countingClicks struct {
	seen []click_usecase.In
}

func (c *countingClicks) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
	c.seen = append(c.seen, in)
	return click_usecase.Out{}, nil
}

type stubReach struct {
	err error
}

func (s stubReach) Refuses(string, uint32) error { return s.err }

func TestAClickOutOfReachGoesNoFurther(t *testing.T) {
	inner := &countingClicks{}

	_, err := reach_click.New(inner, stubReach{err: arenas.ErrOutOfReach}).Execute(t.Context(),
		click_usecase.In{TileID: 7, CountryID: "fr"})

	require.ErrorIs(t, err, arenas.ErrOutOfReach)
	assert.Empty(t, inner.seen, "the throttle inside never sees it, so it spends no token")
}

func TestAClickInReachGoesOn(t *testing.T) {
	inner := &countingClicks{}

	_, err := reach_click.New(inner, stubReach{}).Execute(t.Context(), click_usecase.In{TileID: 7, CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, []click_usecase.In{{TileID: 7, CountryID: "fr"}}, inner.seen)
}
