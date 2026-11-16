package uuid_virus_ids

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses"
)

type Provider struct{}

var _ viruses.IDProvider = Provider{}

func (Provider) NewID() (clicks.VirusID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return clicks.VirusID(id.String()), nil
}
