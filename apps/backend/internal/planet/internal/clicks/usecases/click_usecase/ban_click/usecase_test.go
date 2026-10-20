package ban_click_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/ban_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var until = time.Date(2026, 10, 21, 21, 40, 0, 0, time.UTC)

type bans map[string]time.Time

func (b bans) EnforcedSentence(_ context.Context, scope, account string) (antibot.Sentence, bool) {
	for _, key := range []string{scope, account} {
		if end, ok := b[key]; ok {
			return antibot.Sentence{Until: end}, true
		}
	}
	return antibot.Sentence{}, false
}

type fakeClick struct{ ran bool }

func (c *fakeClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	c.ran = true
	return click_usecase.Out{Outcome: clicks.Taken}, nil
}

func TestABannedScopeIsRefusedAndToldUntilWhen(t *testing.T) {
	inner := &fakeClick{}
	registry := prometheus.NewRegistry()
	useCase := ban_click.New(inner, bans{"2001:db8::/64": until}, registry)

	_, err := useCase.Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), click_usecase.In{TileID: 42, CountryID: "fr"})

	var refusal clicks.BanRefusal
	require.ErrorAs(t, err, &refusal)
	require.ErrorIs(t, err, clicks.ErrBanned)
	assert.Equal(t, until, refusal.Until())
	assert.False(t, inner.ran, "nothing behind the gate runs: no token spent, no watchdog fed")
	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP banned_clicks Clicks refused because their caller is banned
# TYPE banned_clicks counter
banned_clicks 1
`)))
}

func TestABannedAccountIsRefusedFromAnyScope(t *testing.T) {
	inner := &fakeClick{}
	useCase := ban_click.New(inner, bans{"a-guest": until}, prometheus.NewRegistry())

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), "a-guest")
	_, err := useCase.Execute(ctx, click_usecase.In{TileID: 42, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrBanned)
	assert.False(t, inner.ran)
}

func TestAnyoneElseClicks(t *testing.T) {
	inner := &fakeClick{}
	useCase := ban_click.New(inner, bans{"2001:db8::/64": until}, prometheus.NewRegistry())

	out, err := useCase.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), click_usecase.In{TileID: 42, CountryID: "fr"})

	require.NoError(t, err)
	assert.True(t, inner.ran)
	assert.Equal(t, clicks.Taken, out.Outcome)
}
