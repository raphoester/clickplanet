package cpprom_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	metricsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpprom"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	startedAt = time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC)
	readAt    = startedAt.Add(75 * time.Minute)
)

func metricsOf(t *testing.T, registry *prometheus.Registry) (*metricsv1.GetMetricsResponse, error) {
	t.Helper()
	res, err := cpprom.NewAdminService(registry, startedAt, cptime.NewFixedClock(readAt)).
		GetMetrics(t.Context(), connect.NewRequest(&metricsv1.GetMetricsRequest{}))
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the error itself.
	}
	return res.Msg, nil
}

func TestEverySeriesComesBackAsTheRegistryHoldsIt(t *testing.T) {
	registry := prometheus.NewRegistry()
	clicks := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "clicks_total", Help: "h"}, []string{"country_id", "status"})
	banned := prometheus.NewGauge(prometheus.GaugeOpts{Name: "banned_callers", Help: "h"})
	caught := prometheus.NewHistogram(prometheus.HistogramOpts{Name: "bonus_catch_seconds", Help: "h", Buckets: []float64{1, 3}})
	pauses := prometheus.NewSummary(prometheus.SummaryOpts{Name: "gc_seconds", Help: "h", Objectives: map[float64]float64{0.5: 0.05}})
	registry.MustRegister(clicks, banned, caught, pauses)

	clicks.WithLabelValues("fr", "ok").Add(3)
	clicks.WithLabelValues("dz", "error").Inc()
	banned.Set(2)
	caught.Observe(0.5)
	caught.Observe(2)
	pauses.Observe(0.25)

	res, err := metricsOf(t, registry)
	require.NoError(t, err)

	assert.True(t, proto.Equal(&metricsv1.GetMetricsResponse{
		StartedAt: timestamppb.New(startedAt),
		ReadAt:    timestamppb.New(readAt),
		Metrics: []*metricsv1.Metric{
			{Name: "banned_callers", Value: &metricsv1.Metric_Gauge{Gauge: 2}},
			{Name: "bonus_catch_seconds", Value: &metricsv1.Metric_Histogram{Histogram: &metricsv1.Histogram{
				Count: 2, Sum: 2.5, Buckets: []*metricsv1.Bucket{{UpperBound: 1, Count: 1}, {UpperBound: 3, Count: 2}},
			}}},
			{
				Name:   "clicks_total",
				Labels: []*metricsv1.Label{{Name: "country_id", Value: "dz"}, {Name: "status", Value: "error"}},
				Value:  &metricsv1.Metric_Counter{Counter: 1},
			},
			{
				Name:   "clicks_total",
				Labels: []*metricsv1.Label{{Name: "country_id", Value: "fr"}, {Name: "status", Value: "ok"}},
				Value:  &metricsv1.Metric_Counter{Counter: 3},
			},
			{Name: "gc_seconds", Value: &metricsv1.Metric_Summary{Summary: &metricsv1.Summary{
				Count: 1, Sum: 0.25, Quantiles: []*metricsv1.Quantile{{Quantile: 0.5, Value: 0.25}},
			}}},
		},
	}, res), res.String())
}

func TestAnUntypedSeriesIsRefusedRatherThanGuessed(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewUntypedFunc(prometheus.UntypedOpts{Name: "mystery", Help: "h"}, func() float64 { return 1 }))

	_, err := metricsOf(t, registry)

	require.ErrorIs(t, err, cpprom.ErrUnknownType)
}

func TestTheProcessRegistryCarriesTheRuntime(t *testing.T) {
	res, err := metricsOf(t, cpprom.NewRegistry())
	require.NoError(t, err)

	names := cpcolls.NewSet[string]()
	for _, metric := range res.GetMetrics() {
		names.Add(metric.GetName())
	}
	assert.True(t, names.Contains("go_goroutines"), "how busy the process is, beside what it counts")
}
