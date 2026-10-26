package get_metrics_handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrUnknownType = errors.New("a metric of a type the wire cannot carry")

// What /metrics says of the module's own registry, with when its counters started.
func New(gatherer prometheus.Gatherer, startedAt time.Time, clock cptime.Clock) GetMetricsHandler {
	return GetMetricsHandler{gatherer: gatherer, startedAt: startedAt, clock: clock}
}

type GetMetricsHandler struct {
	gatherer  prometheus.Gatherer
	startedAt time.Time
	clock     cptime.Clock
}

func (h GetMetricsHandler) GetMetrics(
	context.Context,
	*connect.Request[planetv1.GetMetricsRequest],
) (*connect.Response[planetv1.GetMetricsResponse], error) {
	readAt := h.clock.Now()

	families, err := h.gatherer.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather the metrics: %w", err)
	}

	res := &planetv1.GetMetricsResponse{StartedAt: timestamppb.New(h.startedAt), ReadAt: timestamppb.New(readAt)}
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

func encode(family *dto.MetricFamily, metric *dto.Metric) (*planetv1.Metric, error) {
	encoded := &planetv1.Metric{Name: family.GetName()}
	for _, label := range metric.GetLabel() {
		encoded.Labels = append(encoded.Labels, &planetv1.Label{Name: label.GetName(), Value: label.GetValue()})
	}

	switch family.GetType() {
	case dto.MetricType_COUNTER:
		encoded.Value = &planetv1.Metric_Counter{Counter: metric.GetCounter().GetValue()}
	case dto.MetricType_GAUGE:
		encoded.Value = &planetv1.Metric_Gauge{Gauge: metric.GetGauge().GetValue()}
	case dto.MetricType_HISTOGRAM:
		histogram := metric.GetHistogram()
		buckets := make([]*planetv1.Bucket, 0, len(histogram.GetBucket()))
		for _, bucket := range histogram.GetBucket() {
			buckets = append(buckets, &planetv1.Bucket{UpperBound: bucket.GetUpperBound(), Count: bucket.GetCumulativeCount()})
		}
		encoded.Value = &planetv1.Metric_Histogram{Histogram: &planetv1.Histogram{
			Count: histogram.GetSampleCount(), Sum: histogram.GetSampleSum(), Buckets: buckets,
		}}
	case dto.MetricType_SUMMARY:
		summary := metric.GetSummary()
		quantiles := make([]*planetv1.Quantile, 0, len(summary.GetQuantile()))
		for _, quantile := range summary.GetQuantile() {
			quantiles = append(quantiles, &planetv1.Quantile{Quantile: quantile.GetQuantile(), Value: quantile.GetValue()})
		}
		encoded.Value = &planetv1.Metric_Summary{Summary: &planetv1.Summary{
			Count: summary.GetSampleCount(), Sum: summary.GetSampleSum(), Quantiles: quantiles,
		}}
	default:
		return nil, fmt.Errorf("%w: %s is %s", ErrUnknownType, family.GetName(), family.GetType())
	}

	return encoded, nil
}
