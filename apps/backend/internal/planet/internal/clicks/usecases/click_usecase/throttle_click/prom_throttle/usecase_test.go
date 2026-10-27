package prom_throttle_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click/prom_throttle"
)

type answering struct {
	out click_usecase.Out
	err error
}

func (a *answering) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return a.out, a.err
}

func counted(clicks int) string {
	return fmt.Sprintf(`
# HELP throttled_clicks Clicks refused because a bucket they spend from was empty
# TYPE throttled_clicks counter
throttled_clicks %d
`, clicks)
}

func TestAThrottledClickIsCountedAndPassedOnAsItCame(t *testing.T) {
	registry := prometheus.NewRegistry()
	refused := click_usecase.Out{Limited: true}
	useCase := prom_throttle.New(&answering{out: refused, err: fmt.Errorf("refused: %w", clicks.ErrThrottled)}, registry)

	out, err := useCase.Execute(t.Context(), click_usecase.In{})

	require.ErrorIs(t, err, clicks.ErrThrottled)
	assert.Equal(t, refused, out, "the budget still goes back to the player")
	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(counted(1))))
}

func TestAnyOtherAnswerIsNotCounted(t *testing.T) {
	registry := prometheus.NewRegistry()
	inner := &answering{out: click_usecase.Out{Outcome: clicks.Taken}}
	useCase := prom_throttle.New(inner, registry)

	_, err := useCase.Execute(t.Context(), click_usecase.In{})
	require.NoError(t, err)
	inner.err = errors.New("the map is down")
	_, err = useCase.Execute(t.Context(), click_usecase.In{})
	require.Error(t, err)

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(counted(0))))
}
