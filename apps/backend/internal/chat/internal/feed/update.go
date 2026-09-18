// Package feed is the chat's live stream: what one client is told, as it happens.
package feed

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

// Update is one frame of the feed: exactly one of a message sent, a message's new reactions and an announcement.
type Update struct {
	Message      *messages.Message
	Reactions    *reactions.Tally
	Announcement *announcements.Announcement
}

// MessageID is the message the update is about, whichever kind it is. An announcement is about no message.
func (u Update) MessageID() messages.MessageID {
	if u.Reactions != nil {
		return u.Reactions.MessageID
	}
	if u.Message != nil {
		return u.Message.ID
	}
	return ""
}
