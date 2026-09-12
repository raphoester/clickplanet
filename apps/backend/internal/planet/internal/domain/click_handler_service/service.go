package click_handler_service

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
)

func New(
	tilesChecker domain.TilesChecker,
	tileStorage domain.TileStorage,
	countryChecker domain.CountryChecker,
) *Service {
	return &Service{
		tilesChecker:   tilesChecker,
		tileStorage:    tileStorage,
		countryChecker: countryChecker,
	}
}

type IService interface {
	HandleClick(ctx context.Context, tileID uint32, countryID string) error
}

type Service struct {
	tilesChecker   domain.TilesChecker
	tileStorage    domain.TileStorage
	countryChecker domain.CountryChecker
}

func (s *Service) HandleClick(ctx context.Context, tileID uint32, countryID string) error {
	if !s.countryChecker.CheckCountry(countryID) {
		return fmt.Errorf("%w: country code %q", domain.ErrInvalidArgument, countryID)
	}

	if !s.tilesChecker.CheckTile(tileID) {
		return fmt.Errorf("%w: tile id %d", domain.ErrInvalidArgument, tileID)
	}

	if err := s.tileStorage.Set(ctx, tileID, countryID); err != nil {
		return fmt.Errorf("failed to set tile: %w", err)
	}

	return nil
}
