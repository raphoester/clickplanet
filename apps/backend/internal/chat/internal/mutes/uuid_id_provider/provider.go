package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

type Provider struct{}

var _ mutes.IDProvider = Provider{}

func (Provider) NewID() (mutes.MuteID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return mutes.MuteID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return mutes.MuteID(id), nil
}
