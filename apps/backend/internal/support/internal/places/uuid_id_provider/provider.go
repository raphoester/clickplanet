package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
)

type Provider struct{}

var _ places.IDProvider = Provider{}

func (Provider) NewID() (places.ReportID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return places.ReportID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return places.ReportID(id), nil
}
