package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Provider struct{}

var _ messages.IDProvider = Provider{}

func (Provider) NewID() (messages.MessageID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate a uuid: %w", err)
	}
	return messages.MessageID(id.String()), nil
}
