package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type Provider struct{}

var _ casts.IDProvider = Provider{}

// Random, not v7: a cast's id goes in a run's public files, and says nothing of when it was made.
func (Provider) NewID() (casts.CastID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return casts.CastID{}, fmt.Errorf("failed to generate a uuid: %w", err)
	}
	return casts.CastID(id), nil
}
