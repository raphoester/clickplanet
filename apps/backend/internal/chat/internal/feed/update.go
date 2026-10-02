package feed

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

type Update struct {
	Message      *messages.Message
	Reactions    *reactions.Tally
	Announcement *announcements.Announcement
}

func (u Update) MessageID() messages.MessageID {
	if u.Reactions != nil {
		return u.Reactions.MessageID
	}
	if u.Message != nil {
		return u.Message.ID
	}
	return ""
}
