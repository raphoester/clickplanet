package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

type Provider struct{}

var _ announcements.IDProvider = Provider{}

func (Provider) NewID() (announcements.AnnouncementID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return announcements.AnnouncementID{}, fmt.Errorf("failed to generate a uuid: %w", err)
	}
	return announcements.AnnouncementID(id), nil
}
