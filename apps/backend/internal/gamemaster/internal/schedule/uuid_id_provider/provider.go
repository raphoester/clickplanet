package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Provider struct{}

var _ schedule.IDProvider = Provider{}

func (Provider) NewID() (schedule.EntryID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return schedule.EntryID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return schedule.EntryID(id), nil
}
