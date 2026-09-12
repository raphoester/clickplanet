package prom_click_handler_service

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func New(
	implementation click_handler_service.IService,
	registerer prometheus.Registerer,
) (*Service, error) {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "clicks",
		Help: "Registered clicks",
	}, []string{
		"source_ip",
		"country_id",
		"status",
	})

	if err := registerer.Register(histogram); err != nil {
		return nil, fmt.Errorf("failed to register histogram: %w", err)
	}

	return &Service{
		histogram:      histogram,
		implementation: implementation,
	}, nil
}

type Service struct {
	implementation click_handler_service.IService
	histogram      *prometheus.HistogramVec
}

func (s *Service) HandleClick(ctx context.Context, tileID uint32, countryID string) error {
	sourceIP := cpctx.GetSourceIP(ctx)
	status := "ok"
	err := s.implementation.HandleClick(ctx, tileID, countryID)
	if err != nil {
		status = "error"
		err = fmt.Errorf("failed to handle click: %w", err)
	}

	s.histogram.WithLabelValues(
		sourceIP,
		countryID,
		status,
	).Observe(1)

	return err
}
