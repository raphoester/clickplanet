package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
)

type Provider struct{}

var _ placements.IDProvider = Provider{}

func (Provider) NewID() (clicks.PlacementID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return clicks.PlacementID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return clicks.PlacementID(id), nil
}
