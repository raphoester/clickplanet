package antibot_click_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/antibot_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fakeGuard struct {
	drop bool

	seen      []antibot.Click
	committed []antibot.Click
}

func (g *fakeGuard) Inspect(click antibot.Click) bool {
	g.seen = append(g.seen, click)
	return g.drop
}

func (g *fakeGuard) Committed(click antibot.Click) {
	g.committed = append(g.committed, click)
}

func (g *fakeGuard) Flagged() int { return len(g.seen) }

type fakeOwner map[uint32]string

func (o fakeOwner) Owner(tile uint32) (string, bool) {
	held, ok := o[tile]
	return held, ok
}

type fakeClick struct {
	err error
	ran bool
}

func (c *fakeClick) Execute(context.Context, click.In) (click.Out, error) {
	c.ran = true
	return click.Out{}, c.err
}

func execute(
	t *testing.T,
	ctx context.Context,
	guard antibot_click.ClickGuard,
	owner antibot_click.TileOwner,
	inner *fakeClick,
) error {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	useCase, err := antibot_click.New(inner, guard, owner, clock, prometheus.NewRegistry())
	require.NoError(t, err)

	_, executeErr := useCase.Execute(ctx, click.In{TileID: 42, CountryID: "PS"})

	return executeErr
}

func TestAntiBotClick(t *testing.T) {
	t.Run("lets an unflagged click through to the map", func(t *testing.T) {
		guard := &fakeGuard{drop: false}
		inner := &fakeClick{}

		require.NoError(t, execute(t, t.Context(), guard, fakeOwner{}, inner))
		require.True(t, inner.ran)
		require.Len(t, guard.committed, 1, "a click the use case accepted reached the map")
	})

	t.Run("answers a flagged click OK without touching the map", func(t *testing.T) {
		guard := &fakeGuard{drop: true}
		inner := &fakeClick{}

		require.NoError(t, execute(t, t.Context(), guard, fakeOwner{}, inner),
			"a shadow ban must look exactly like success")
		require.False(t, inner.ran, "the map must not be touched")
		require.Empty(t, guard.committed, "a dropped click took no tile")
	})

	t.Run("does not report a click the use case refused", func(t *testing.T) {
		guard := &fakeGuard{}
		inner := &fakeClick{err: errors.New("nope")}

		require.Error(t, execute(t, t.Context(), guard, fakeOwner{}, inner))
		require.Len(t, guard.seen, 1)
		require.Empty(t, guard.committed, "a refused click changed no tile")
	})

	t.Run("charges the same scope the throttle is charged to", func(t *testing.T) {
		guard := &fakeGuard{}
		ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::dead:beef")

		require.NoError(t, execute(t, ctx, guard, fakeOwner{}, &fakeClick{}))
		require.Len(t, guard.seen, 1)
		assert.Equal(t, "2001:db8::/64", guard.seen[0].Scope)
	})

	t.Run("reads who held the tile before the write could change it", func(t *testing.T) {
		guard := &fakeGuard{}

		require.NoError(t, execute(t, t.Context(), guard, fakeOwner{42: "FR"}, &fakeClick{}))
		require.Len(t, guard.seen, 1)
		assert.Equal(t, "FR", guard.seen[0].Held)
		assert.False(t, guard.seen[0].NoOp, "PS taking a tile FR holds changes the map")
	})

	t.Run("marks a click onto a tile the caller's own country holds", func(t *testing.T) {
		guard := &fakeGuard{}

		require.NoError(t, execute(t, t.Context(), guard, fakeOwner{42: "PS"}, &fakeClick{}))
		require.Len(t, guard.seen, 1)
		assert.True(t, guard.seen[0].NoOp, "it changes nothing and publishes nothing")
	})
}
