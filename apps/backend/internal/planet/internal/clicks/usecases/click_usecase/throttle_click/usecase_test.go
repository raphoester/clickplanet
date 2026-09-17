package throttle_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type fakeLimiter struct {
	allow bool
	state cpratelimit.State
	keys  [][]cpratelimit.Key
	spent []float64
}

// TakeAll answers state for every key.
func (l *fakeLimiter) TakeAll(n float64, keys ...cpratelimit.Key) (bool, []cpratelimit.State) {
	l.keys = append(l.keys, keys)
	l.spent = append(l.spent, n)

	states := make([]cpratelimit.State, len(keys))
	for i := range states {
		states[i] = l.state
	}
	return l.allow, states
}

type fakePricer struct {
	price     clicks.Price
	countries []string
}

func (p *fakePricer) Price(country string) clicks.Price {
	p.countries = append(p.countries, country)
	return p.price
}

func onePrice() *fakePricer { return &fakePricer{price: clicks.Price{Cost: 1}} }

var buckets = clicks.ThrottleConfig{}.Buckets()

type fakeClick struct {
	err error
	ran bool
	in  click_usecase.In
}

func (c *fakeClick) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
	c.ran, c.in = true, in
	return click_usecase.Out{}, c.err
}

func TestThrottleClick(t *testing.T) {
	state := cpratelimit.State{Tokens: 4, Capacity: 5, PerSecond: 2}

	t.Run("lets an allowed click through and says what is left", func(t *testing.T) {
		inner := &fakeClick{}

		out, err := throttle_click.New(inner, &fakeLimiter{allow: true, state: state}, onePrice(), buckets).
			Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "fr"})

		require.NoError(t, err)
		require.True(t, inner.ran)
		assert.True(t, out.Limited)
		assert.Equal(t, state, out.Budget.State)
	})

	t.Run("marks a click boosted while the bucket says a boost runs", func(t *testing.T) {
		plain, boosted := &fakeClick{}, &fakeClick{}

		_, err := throttle_click.New(plain, &fakeLimiter{allow: true, state: state}, onePrice(), buckets).
			Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "fr"})
		require.NoError(t, err)

		boostedState := state
		boostedState.Boosted = true
		_, err = throttle_click.New(boosted, &fakeLimiter{allow: true, state: boostedState}, onePrice(), buckets).
			Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "fr"})
		require.NoError(t, err)

		assert.False(t, plain.in.Boosted)
		assert.True(t, boosted.in.Boosted)
	})

	t.Run("refuses a click over the limit without touching the map", func(t *testing.T) {
		inner := &fakeClick{}

		out, err := throttle_click.New(inner, &fakeLimiter{allow: false, state: state}, onePrice(), buckets).
			Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "fr"})

		require.ErrorIs(t, err, clicks.ErrThrottled)
		require.False(t, inner.ran, "a refused click must not reach the map")
		assert.True(t, out.Limited, "the refusal is the answer that most needs the reading")
		assert.Equal(t, state, out.Budget.State)
	})

	t.Run("keys on the scope a budget is charged to, not the address", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::dead:beef")

		_, err := throttle_click.New(&fakeClick{}, limiter, onePrice(), buckets).Execute(ctx, click_usecase.In{})

		require.NoError(t, err)
		assert.Equal(t, [][]cpratelimit.Key{{{Name: "2001:db8::/64", Scale: 1}}}, limiter.keys,
			"a token with no account spends the scope's bucket alone, as before accounts")
	})

	t.Run("charges the account and its scope at ten times the account's allowance", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

		_, err := throttle_click.New(&fakeClick{}, limiter, onePrice(), buckets).Execute(ctx, click_usecase.In{})

		require.NoError(t, err)
		assert.Equal(t, [][]cpratelimit.Key{{
			{Name: "account:a-guest", Scale: 1},
			{Name: "scope:1.2.3.4", Scale: 10},
		}}, limiter.keys)
	})

	t.Run("still reports the allowance when the click itself failed", func(t *testing.T) {
		inner := &fakeClick{err: errors.New("disk on fire")}

		out, err := throttle_click.New(inner, &fakeLimiter{allow: true, state: state}, onePrice(), buckets).
			Execute(t.Context(), click_usecase.In{})

		require.Error(t, err)
		assert.True(t, out.Limited, "the token was spent, so the meter has to move")
		assert.Equal(t, state, out.Budget.State)
	})

	t.Run("charges the price of the country clicked for, and counts what is left in clicks", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true, state: cpratelimit.State{Tokens: 6, Capacity: 10, PerSecond: 1}}
		pricer := &fakePricer{price: clicks.Price{Cost: 1.5, Share: 0.4}}

		out, err := throttle_click.New(&fakeClick{}, limiter, pricer, buckets).
			Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "bg"})

		require.NoError(t, err)
		assert.Equal(t, []string{"bg"}, pricer.countries)
		assert.Equal(t, []float64{1.5}, limiter.spent)
		assert.InDelta(t, 4.0, out.Budget.Tokens, 1e-9)
		assert.Equal(t, 6, out.Budget.Capacity, "6.67 clicks of room is six whole ones")
		assert.InDelta(t, 1.5, out.Budget.Price.Cost, 1e-9)
	})
}
