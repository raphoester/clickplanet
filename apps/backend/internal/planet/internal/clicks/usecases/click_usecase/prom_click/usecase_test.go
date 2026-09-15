package prom_click_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/prom_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type fakeClick struct{ err error }

func (c *fakeClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{}, c.err
}

func TestEveryClickIsCountedByCountryAndOutcomeAndNeverByAddress(t *testing.T) {
	registry := prometheus.NewRegistry()
	inner := &fakeClick{}
	usecase := prom_click.New(inner, registry)
	ctx := cpctx.AddIPToContext(t.Context(), "198.51.100.20")

	_, err := usecase.Execute(ctx, click_usecase.In{CountryID: "fr"})
	require.NoError(t, err)
	_, err = usecase.Execute(ctx, click_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	inner.err = errors.New("boom")
	_, err = usecase.Execute(ctx, click_usecase.In{CountryID: "de"})
	assert.ErrorIs(t, err, inner.err)

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP clicks_total Clicks that reached the rule, by country and outcome
# TYPE clicks_total counter
clicks_total{country_id="de",status="error"} 1
clicks_total{country_id="fr",status="ok"} 2
`)))
}
