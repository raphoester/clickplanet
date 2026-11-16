package uuid_rally_ids

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
)

type Provider struct{}

var _ rallies.IDProvider = Provider{}

func (Provider) NewID() (rallies.RallyID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return rallies.RallyID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return rallies.RallyID(id), nil
}
