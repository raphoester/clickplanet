// Package chatannouncement puts an announcement on the wire.
package chatannouncement

import (
	"github.com/google/uuid"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

func Encode(announcement announcements.Announcement) *chatv1.Announcement {
	return &chatv1.Announcement{
		Id:                uuid.UUID(announcement.ID).String(),
		AnnouncedAtUnixMs: announcement.At.UnixMilli(),
		Kind:              string(announcement.Kind),
		Payload:           string(announcement.Payload),
	}
}
