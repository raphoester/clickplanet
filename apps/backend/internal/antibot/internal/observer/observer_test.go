package observer_test

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/observer"
)

func TestOpinionsAreCountedPerWatchdogAndLevel(t *testing.T) {
	registry := prometheus.NewRegistry()
	observed := observer.New(slog.New(slog.DiscardHandler), registry)

	observed.OnRise("sequencer", "suspect")
	observed.OnRise("sequencer", "suspect")
	observed.OnRise("metronome", "certain")

	observed.OnStanding("sequencer", "suspect", 4)
	observed.OnStanding("sequencer", "suspect", 1)

	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP antibot_opinions_total Times a watchdog's reading of a caller rose to a level it had not held within jury.suspicionWindow; suspect includes certain
# TYPE antibot_opinions_total counter
antibot_opinions_total{level="certain",watchdog="metronome"} 1
antibot_opinions_total{level="suspect",watchdog="sequencer"} 2
# HELP antibot_opinions_standing Callers a watchdog reads at a level or above at the last jury sweep; suspect includes certain
# TYPE antibot_opinions_standing gauge
antibot_opinions_standing{level="suspect",watchdog="sequencer"} 1
`), "antibot_opinions_total", "antibot_opinions_standing"))
}

type flagged struct {
	running int
	err     error
}

func (f flagged) Flagged(context.Context) (int, error) { return f.running, f.err }

func TestTheRunningBansAreCountedAtEachScrape(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer.CountBans(flagged{running: 3}, registry)

	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP banned_callers Callers currently banned, whether or not bans.enforce is on
# TYPE banned_callers gauge
banned_callers 3
`)))
}

func TestBansThatCannotBeCountedAreNoNumber(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer.CountBans(flagged{err: errors.New("postgres is down")}, registry)

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	assert.True(t, math.IsNaN(families[0].GetMetric()[0].GetGauge().GetValue()))
}
