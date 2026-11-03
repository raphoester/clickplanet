package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
)

type Provider struct{}

var _ grants.IDProvider = Provider{}

func (Provider) NewID() (grants.GrantID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return grants.GrantID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return grants.GrantID(id), nil
}
