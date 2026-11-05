package refusal_click_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/refusal_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at    = time.Date(2026, 10, 21, 20, 40, 0, 0, time.UTC)
	until = at.Add(time.Hour)
)

const account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

func refusing(t *testing.T, callers ...refusals.Caller) (refusals.Book, *refusals.MemoryStore) {
	t.Helper()
	store := refusals.NewMemoryStore()
	book := refusals.NewBook(store, cptime.NewFixedClock(at))
	for _, caller := range callers {
		require.NoError(t, book.Decide(t.Context(), caller, refusals.Refused(until, at)))
	}
	return book, store
}

type fakeClick struct{ ran bool }

func (c *fakeClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	c.ran = true
	return click_usecase.Out{Outcome: clicks.Taken}, nil
}

func TestARefusedScopeIsRefusedAndToldUntilWhen(t *testing.T) {
	inner := &fakeClick{}
	registry := prometheus.NewRegistry()
	book, _ := refusing(t, refusals.Caller{Scope: "2001:db8::/64"})

	_, err := refusal_click.New(inner, book, registry).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), click_usecase.In{TileID: 42, CountryID: "fr"})

	var refusal clicks.Refusal
	require.ErrorAs(t, err, &refusal)
	require.ErrorIs(t, err, clicks.ErrRefused)
	assert.Equal(t, until, refusal.Until())
	assert.False(t, inner.ran, "nothing behind the gate runs: no token spent")
	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP refused_clicks Clicks refused because their caller is refused
# TYPE refused_clicks counter
refused_clicks 1
`)))
}

func TestARefusedAccountIsRefusedFromAnyScope(t *testing.T) {
	inner := &fakeClick{}
	book, _ := refusing(t, refusals.Caller{Account: account})

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), account)
	_, err := refusal_click.New(inner, book, prometheus.NewRegistry()).Execute(ctx, click_usecase.In{TileID: 42, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrRefused)
	assert.False(t, inner.ran)
}

func TestAnyoneElseClicks(t *testing.T) {
	inner := &fakeClick{}
	book, _ := refusing(t, refusals.Caller{Scope: "2001:db8::/64"})

	out, err := refusal_click.New(inner, book, prometheus.NewRegistry()).
		Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), click_usecase.In{TileID: 42, CountryID: "fr"})

	require.NoError(t, err)
	assert.True(t, inner.ran)
	assert.Equal(t, clicks.Taken, out.Outcome)
}

func TestARefusalThatCannotBeReadLetsTheClickThrough(t *testing.T) {
	inner := &fakeClick{}
	book, store := refusing(t, refusals.Caller{Scope: "203.0.113.7"})
	store.FailWith(errors.New("postgres is down"))

	_, err := refusal_click.New(inner, book, prometheus.NewRegistry()).
		Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), click_usecase.In{TileID: 42, CountryID: "fr"})

	require.NoError(t, err)
	assert.True(t, inner.ran)
}
