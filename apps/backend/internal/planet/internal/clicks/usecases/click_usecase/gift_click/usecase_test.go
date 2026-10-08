package gift_click_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/gift_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/inmemory_gift_cache"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/inmemory_gift_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type stubClick struct{ err error }

func (s stubClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{}, s.err
}

type game struct {
	switches *tempo.Switches
	given    *inmemory_gift_storage.Storage
	limiter  *cpratelimit.Limiter
	charges  *inmemory_charge_storage.Storage
	buckets  clicks.Buckets
}

func newGame(t *testing.T) game {
	t.Helper()

	return game{
		switches: tempo.NewSwitches(),
		given:    inmemory_gift_storage.New(),
		limiter:  cpratelimit.New("test", cpratelimit.Config{PerSecond: 0.2, Burst: 60}, cptime.NewFixedClock(finale)),
		charges: inmemory_charge_storage.New(inmemory_charge_storage.Config{}, bonuses.ChargesConfig{},
			inmemory_charge_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler)),
		buckets: clicks.ThrottleConfig{}.Buckets(),
	}
}

func (g game) giving(t *testing.T, madeBefore time.Time) game {
	t.Helper()

	gift, err := tempo.GiftOf("finale-0", madeBefore)
	require.NoError(t, err)
	rules, err := tempo.NewRules(3, 2*time.Minute, false)
	require.NoError(t, err)
	g.switches.Set(rules.WithGift(gift))
	return g
}

func (g game) click(ctx context.Context, inner click_usecase.IUseCase) (click_usecase.Out, error) {
	return gift_click.New(inner, g.switches, inmemory_gift_cache.New(g.given), g.limiter, g.charges, g.buckets).
		Execute(ctx, click_usecase.In{TileID: 1, CountryID: "fr"})
}

func guest(t *testing.T, account string, created time.Time) context.Context {
	t.Helper()

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), account)
	return cpctx.AddAccountCreatedToContext(ctx, created)
}

func (g game) spend(keys []cpratelimit.Key, n float64) {
	g.limiter.TakeAll(n, keys...)
}

func TestTheFirstAcceptedClickOfAFinaleFillsTheBankAndGrantsABomb(t *testing.T) {
	g := newGame(t).giving(t, finale)
	ctx := guest(t, "ada", finale.Add(-time.Hour))
	payer := clicks.PayerOf(ctx)
	g.spend(g.buckets.Keys(payer, clicks.Price{}), 50)

	out, err := g.click(ctx, stubClick{})

	require.NoError(t, err)
	assert.True(t, out.Gift)
	assert.True(t, g.charges.Held("ada").Bomb)
	keys := g.buckets.Keys(payer, clicks.Price{})
	assert.InDelta(t, 60.0, g.limiter.Peek(keys[0]).Tokens, 1e-9, "the account's own bucket is full")
	assert.InDelta(t, 60.0, g.limiter.Peek(keys[1]).Tokens, 1e-9, "the guests' bucket is full, or a guest's bank would hold nothing")
	assert.InDelta(t, 550.0, g.limiter.Peek(keys[2]).Tokens, 1e-9, "the scope's bucket is everybody's and is never filled")
}

func TestTheGiftIsGivenOnce(t *testing.T) {
	g := newGame(t).giving(t, finale)
	ctx := guest(t, "ada", finale.Add(-time.Hour))

	_, err := g.click(ctx, stubClick{})
	require.NoError(t, err)
	require.True(t, g.charges.SpendBomb("ada"))

	out, err := g.click(ctx, stubClick{})

	require.NoError(t, err)
	assert.False(t, out.Gift)
	assert.False(t, g.charges.Held("ada").Bomb)
}

func TestTheGiftIsGivenOnceAcrossARestart(t *testing.T) {
	g := newGame(t).giving(t, finale)
	ctx := guest(t, "ada", finale.Add(-time.Hour))
	_, err := g.click(ctx, stubClick{})
	require.NoError(t, err)

	restarted := newGame(t).giving(t, finale)
	restarted.given = g.given

	out, err := restarted.click(ctx, stubClick{})

	require.NoError(t, err)
	assert.False(t, out.Gift)
	assert.False(t, restarted.charges.Held("ada").Bomb)
}

func TestARefusedClickGetsNoGift(t *testing.T) {
	g := newGame(t).giving(t, finale)

	out, err := g.click(guest(t, "ada", finale.Add(-time.Hour)), stubClick{err: clicks.ErrUnknownCountry})

	require.ErrorIs(t, err, clicks.ErrUnknownCountry)
	assert.False(t, out.Gift)
	assert.Empty(t, g.given.Given("finale-0"), "a later accepted click still gets it")
}

func TestADroppedClickGetsTheGiftLikeAnyOther(t *testing.T) {
	g := newGame(t).giving(t, finale)

	out, err := g.click(guest(t, "ada", finale.Add(-time.Hour)), stubClick{})

	require.NoError(t, err)
	assert.True(t, out.Gift, "the shadow ban answers a dropped click with the zero Out and no error")
}

func TestNoGiftIsGivenOutsideAFinale(t *testing.T) {
	g := newGame(t)

	out, err := g.click(guest(t, "ada", finale.Add(-time.Hour)), stubClick{})

	require.NoError(t, err)
	assert.False(t, out.Gift)
	assert.False(t, g.charges.Held("ada").Bomb)
}

func TestAnAccountMadeDuringTheFinaleGetsNoGift(t *testing.T) {
	g := newGame(t).giving(t, finale)

	out, err := g.click(guest(t, "fresh", finale.Add(time.Minute)), stubClick{})

	require.NoError(t, err)
	assert.False(t, out.Gift)
	assert.False(t, g.charges.Held("fresh").Bomb)
}

func TestACallerWithNoAccountGetsNoGift(t *testing.T) {
	g := newGame(t).giving(t, finale)

	out, err := g.click(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), stubClick{})

	require.NoError(t, err)
	assert.False(t, out.Gift)
}

func TestAGiftThatCannotBeRecordedIsNotGivenAndStaysOwed(t *testing.T) {
	g := newGame(t).giving(t, finale)
	ctx := guest(t, "ada", finale.Add(-time.Hour))
	g.given.FailWith(errors.New("postgres is down"))

	out, err := g.click(ctx, stubClick{})

	require.NoError(t, err, "the tile was taken; a gift is no reason to refuse it")
	assert.False(t, out.Gift)
	assert.False(t, g.charges.Held("ada").Bomb)

	g.given.FailWith(nil)
	out, err = g.click(ctx, stubClick{})
	require.NoError(t, err)
	assert.True(t, out.Gift)
}
