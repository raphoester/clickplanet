package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

type Provider struct{}

var _ graffiti.IDProvider = Provider{}

func (Provider) NewID() (graffiti.ID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return graffiti.ID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return graffiti.ID(id), nil
}
