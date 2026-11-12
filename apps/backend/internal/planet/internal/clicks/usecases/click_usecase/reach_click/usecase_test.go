package reach_click_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
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

func (s stubReach) Refuses(string, uint32, int) error { return s.err }

func TestAClickOutOfReachGoesNoFurther(t *testing.T) {
	inner := &countingClicks{}

	_, err := reach_click.New(inner, stubReach{err: clicks.ErrOutOfReach}).Execute(t.Context(),
		click_usecase.In{TileID: 7, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrOutOfReach)
	assert.Empty(t, inner.seen, "the throttle inside never sees it, so it spends no token")
}

func TestAClickInReachGoesOn(t *testing.T) {
	inner := &countingClicks{}

	_, err := reach_click.New(inner, stubReach{}).Execute(t.Context(), click_usecase.In{TileID: 7, CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, []click_usecase.In{{TileID: 7, CountryID: "fr"}}, inner.seen)
}

type ringsAsked struct{ rings []int }

func (r *ringsAsked) Refuses(_ string, _ uint32, rings int) error {
	r.rings = append(r.rings, rings)
	return nil
}

func TestAClickThatSpreadsIsCheckedWithItsRings(t *testing.T) {
	asked := &ringsAsked{}

	_, err := reach_click.New(&countingClicks{}, asked).Execute(t.Context(),
		click_usecase.In{TileID: 7, CountryID: "fr", Spread: true, Rings: 2})

	require.NoError(t, err)
	assert.Equal(t, []int{2}, asked.rings)
}
