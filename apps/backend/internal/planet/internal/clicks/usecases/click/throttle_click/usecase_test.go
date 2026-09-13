package throttle_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type fakeLimiter struct {
	allow bool
	state cpratelimit.State
	keys  []string
	spent []int
}

func (l *fakeLimiter) TakeN(key string, n int) (bool, cpratelimit.State) {
	l.keys = append(l.keys, key)
	l.spent = append(l.spent, n)
	return l.allow, l.state
}

type fakePricer struct {
	price     toll.Price
	countries []string
}

func (p *fakePricer) Price(country string) toll.Price {
	p.countries = append(p.countries, country)
	return p.price
}

func onePrice() *fakePricer { return &fakePricer{price: toll.Price{Cost: 1}} }

type fakeClick struct {
	err error
	ran bool
}

func (c *fakeClick) Execute(context.Context, click.In) (click.Out, error) {
	c.ran = true
	return click.Out{}, c.err
}

func TestThrottleClick(t *testing.T) {
	state := cpratelimit.State{Tokens: 4, Capacity: 5, PerSecond: 2}

	t.Run("lets an allowed click through and says what is left", func(t *testing.T) {
		inner := &fakeClick{}

		out, err := throttle_click.New(inner, &fakeLimiter{allow: true, state: state}, onePrice()).
			Execute(t.Context(), click.In{TileID: 1, CountryID: "fr"})

		require.NoError(t, err)
		require.True(t, inner.ran)
		assert.True(t, out.Limited)
		assert.Equal(t, state, out.Budget.State)
	})

	t.Run("refuses a click over the limit without touching the map", func(t *testing.T) {
		inner := &fakeClick{}

		out, err := throttle_click.New(inner, &fakeLimiter{allow: false, state: state}, onePrice()).
			Execute(t.Context(), click.In{TileID: 1, CountryID: "fr"})

		require.ErrorIs(t, err, clicks.ErrThrottled)
		require.False(t, inner.ran, "a refused click must not reach the map")
		assert.True(t, out.Limited, "the refusal is the answer that most needs the reading")
		assert.Equal(t, state, out.Budget.State)
	})

	t.Run("keys on the scope a budget is charged to, not the address", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::dead:beef")

		_, err := throttle_click.New(&fakeClick{}, limiter, onePrice()).Execute(ctx, click.In{})

		require.NoError(t, err)
		assert.Equal(t, []string{"2001:db8::/64"}, limiter.keys)
	})

	t.Run("still reports the allowance when the click itself failed", func(t *testing.T) {
		inner := &fakeClick{err: errors.New("disk on fire")}

		out, err := throttle_click.New(inner, &fakeLimiter{allow: true, state: state}, onePrice()).
			Execute(t.Context(), click.In{})

		require.Error(t, err)
		assert.True(t, out.Limited, "the token was spent, so the meter has to move")
		assert.Equal(t, state, out.Budget.State)
	})

	t.Run("charges the price of the country clicked for, and counts what is left in clicks", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true, state: cpratelimit.State{Tokens: 6, Capacity: 10, PerSecond: 1}}
		pricer := &fakePricer{price: toll.Price{Cost: 3, Share: 0.4}}

		out, err := throttle_click.New(&fakeClick{}, limiter, pricer).
			Execute(t.Context(), click.In{TileID: 1, CountryID: "bg"})

		require.NoError(t, err)
		assert.Equal(t, []string{"bg"}, pricer.countries)
		assert.Equal(t, []int{3}, limiter.spent)
		assert.InDelta(t, 2.0, out.Budget.Tokens, 1e-9)
		assert.Equal(t, 3, out.Budget.Capacity)
		assert.Equal(t, 3, out.Budget.Price.Cost)
	})
}
