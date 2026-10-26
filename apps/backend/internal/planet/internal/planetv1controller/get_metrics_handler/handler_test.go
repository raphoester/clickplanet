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

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_metrics_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	startedAt = time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC)
	readAt    = startedAt.Add(75 * time.Minute)
)

func metricsOf(t *testing.T, registry *prometheus.Registry) (*planetv1.GetMetricsResponse, error) {
	t.Helper()
	res, err := get_metrics_handler.New(registry, startedAt, cptime.NewFixedClock(readAt)).
		GetMetrics(t.Context(), connect.NewRequest(&planetv1.GetMetricsRequest{}))
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the error itself.
	}
	return res.Msg, nil
}

func TestEverySeriesComesBackAsTheRegistryHoldsIt(t *testing.T) {
	registry := prometheus.NewRegistry()
	clicks := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "clicks_total", Help: "h"}, []string{"country_id", "status"})
	running := prometheus.NewGauge(prometheus.GaugeOpts{Name: "boxes_waiting", Help: "h"})
	caught := prometheus.NewHistogram(prometheus.HistogramOpts{Name: "bonus_catch_seconds", Help: "h", Buckets: []float64{1, 3}})
	pauses := prometheus.NewSummary(prometheus.SummaryOpts{Name: "pause_seconds", Help: "h", Objectives: map[float64]float64{0.5: 0.05}})
	registry.MustRegister(clicks, running, caught, pauses)

	clicks.WithLabelValues("fr", "ok").Add(3)
	clicks.WithLabelValues("dz", "error").Inc()
	running.Set(2)
	caught.Observe(0.5)
	caught.Observe(2)
	pauses.Observe(0.25)

	res, err := metricsOf(t, registry)
	require.NoError(t, err)

	assert.True(t, proto.Equal(&planetv1.GetMetricsResponse{
		StartedAt: timestamppb.New(startedAt),
		ReadAt:    timestamppb.New(readAt),
		Metrics: []*planetv1.Metric{
			{Name: "bonus_catch_seconds", Value: &planetv1.Metric_Histogram{Histogram: &planetv1.Histogram{
				Count: 2, Sum: 2.5, Buckets: []*planetv1.Bucket{{UpperBound: 1, Count: 1}, {UpperBound: 3, Count: 2}},
			}}},
			{Name: "boxes_waiting", Value: &planetv1.Metric_Gauge{Gauge: 2}},
			{
				Name:   "clicks_total",
				Labels: []*planetv1.Label{{Name: "country_id", Value: "dz"}, {Name: "status", Value: "error"}},
				Value:  &planetv1.Metric_Counter{Counter: 1},
			},
			{
				Name:   "clicks_total",
				Labels: []*planetv1.Label{{Name: "country_id", Value: "fr"}, {Name: "status", Value: "ok"}},
				Value:  &planetv1.Metric_Counter{Counter: 3},
			},
			{Name: "pause_seconds", Value: &planetv1.Metric_Summary{Summary: &planetv1.Summary{
				Count: 1, Sum: 0.25, Quantiles: []*planetv1.Quantile{{Quantile: 0.5, Value: 0.25}},
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
