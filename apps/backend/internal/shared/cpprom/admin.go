package cpprom

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/types/known/timestamppb"

	metricsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrUnknownType = errors.New("a metric of a type the wire cannot carry")

// What /metrics says, for the admin page: every series the registry gathers, with when its counters started.
func NewAdminService(gatherer prometheus.Gatherer, startedAt time.Time, clock cptime.Clock) AdminService {
	return AdminService{gatherer: gatherer, startedAt: startedAt, clock: clock}
}

type AdminService struct {
	gatherer  prometheus.Gatherer
	startedAt time.Time
	clock     cptime.Clock
}

func (s AdminService) GetMetrics(
	context.Context,
	*connect.Request[metricsv1.GetMetricsRequest],
) (*connect.Response[metricsv1.GetMetricsResponse], error) {
	readAt := s.clock.Now()

	families, err := s.gatherer.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather the metrics: %w", err)
	}

	res := &metricsv1.GetMetricsResponse{StartedAt: timestamppb.New(s.startedAt), ReadAt: timestamppb.New(readAt)}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			encoded, err := encode(family, metric)
			if err != nil {
				return nil, err
			}
			res.Metrics = append(res.Metrics, encoded)
		}
	}

	return connect.NewResponse(res), nil
}

func encode(family *dto.MetricFamily, metric *dto.Metric) (*metricsv1.Metric, error) {
	encoded := &metricsv1.Metric{Name: family.GetName()}
	for _, label := range metric.GetLabel() {
		encoded.Labels = append(encoded.Labels, &metricsv1.Label{Name: label.GetName(), Value: label.GetValue()})
	}

	switch family.GetType() {
	case dto.MetricType_COUNTER:
		encoded.Value = &metricsv1.Metric_Counter{Counter: metric.GetCounter().GetValue()}
	case dto.MetricType_GAUGE:
		encoded.Value = &metricsv1.Metric_Gauge{Gauge: metric.GetGauge().GetValue()}
	case dto.MetricType_HISTOGRAM:
		histogram := metric.GetHistogram()
		buckets := make([]*metricsv1.Bucket, 0, len(histogram.GetBucket()))
		for _, bucket := range histogram.GetBucket() {
			buckets = append(buckets, &metricsv1.Bucket{UpperBound: bucket.GetUpperBound(), Count: bucket.GetCumulativeCount()})
		}
		encoded.Value = &metricsv1.Metric_Histogram{Histogram: &metricsv1.Histogram{
			Count: histogram.GetSampleCount(), Sum: histogram.GetSampleSum(), Buckets: buckets,
		}}
	case dto.MetricType_SUMMARY:
		summary := metric.GetSummary()
		quantiles := make([]*metricsv1.Quantile, 0, len(summary.GetQuantile()))
		for _, quantile := range summary.GetQuantile() {
			quantiles = append(quantiles, &metricsv1.Quantile{Quantile: quantile.GetQuantile(), Value: quantile.GetValue()})
		}
		encoded.Value = &metricsv1.Metric_Summary{Summary: &metricsv1.Summary{
			Count: summary.GetSampleCount(), Sum: summary.GetSampleSum(), Quantiles: quantiles,
		}}
	default:
		return nil, fmt.Errorf("%w: %s is %s", ErrUnknownType, family.GetName(), family.GetType())
	}

	return encoded, nil
}
