package get_metrics_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_metrics_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	startedAt = time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC)
	readAt    = startedAt.Add(75 * time.Minute)
)

func metricsOf(t *testing.T, registry *prometheus.Registry) (*antibotv1.GetMetricsResponse, error) {
	t.Helper()
	res, err := get_metrics_handler.New(registry, startedAt, cptime.NewFixedClock(readAt)).
		GetMetrics(t.Context(), connect.NewRequest(&antibotv1.GetMetricsRequest{}))
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the error itself.
	}
	return res.Msg, nil
}

func TestEverySeriesComesBackAsTheRegistryHoldsIt(t *testing.T) {
	registry := prometheus.NewRegistry()
	flags := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ban_flags", Help: "h"}, []string{"watchdog"})
	running := prometheus.NewGauge(prometheus.GaugeOpts{Name: "banned_callers", Help: "h"})
	reactions := prometheus.NewHistogram(prometheus.HistogramOpts{Name: "click_reaction_seconds", Help: "h", Buckets: []float64{1, 3}})
	registry.MustRegister(flags, running, reactions)

	flags.WithLabelValues("catcher").Add(3)
	flags.WithLabelValues("sequencer").Inc()
	running.Set(2)
	reactions.Observe(0.5)
	reactions.Observe(2)

	res, err := metricsOf(t, registry)
	require.NoError(t, err)

	assert.True(t, proto.Equal(&antibotv1.GetMetricsResponse{
		StartedAt: timestamppb.New(startedAt),
		ReadAt:    timestamppb.New(readAt),
		Metrics: []*antibotv1.Metric{
			{
				Name:   "ban_flags",
				Labels: []*antibotv1.Label{{Name: "watchdog", Value: "catcher"}},
				Value:  &antibotv1.Metric_Counter{Counter: 3},
			},
			{
				Name:   "ban_flags",
				Labels: []*antibotv1.Label{{Name: "watchdog", Value: "sequencer"}},
				Value:  &antibotv1.Metric_Counter{Counter: 1},
			},
			{Name: "banned_callers", Value: &antibotv1.Metric_Gauge{Gauge: 2}},
			{Name: "click_reaction_seconds", Value: &antibotv1.Metric_Histogram{Histogram: &antibotv1.Histogram{
				Count: 2, Sum: 2.5, Buckets: []*antibotv1.Bucket{{UpperBound: 1, Count: 1}, {UpperBound: 3, Count: 2}},
			}}},
		},
	}, res), res.String())
}

func TestAnUntypedSeriesIsRefusedRatherThanGuessed(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewUntypedFunc(prometheus.UntypedOpts{Name: "mystery", Help: "h"}, func() float64 { return 1 }))

	_, err := metricsOf(t, registry)

	require.ErrorIs(t, err, get_metrics_handler.ErrUnknownType)
}
