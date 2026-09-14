package antibot_click_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/antibot_click"
)

func TestOpinionsAreCountedPerWatchdogAndLevel(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := antibot_click.NewObserver(slog.New(slog.DiscardHandler), registry)

	observer.OnRise("sequencer", "suspect")
	observer.OnRise("sequencer", "suspect")
	observer.OnRise("metronome", "certain")

	observer.OnStanding("sequencer", "suspect", 4)
	observer.OnStanding("sequencer", "suspect", 1)

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
