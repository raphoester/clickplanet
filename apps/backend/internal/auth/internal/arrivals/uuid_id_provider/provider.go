package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
)

type Provider struct{}

var _ arrivals.IDProvider = Provider{}

func (Provider) NewID() (arrivals.ArrivalID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return arrivals.ArrivalID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return arrivals.ArrivalID(id), nil
}
