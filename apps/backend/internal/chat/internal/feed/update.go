package feed

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

type Update struct {
	message      *messages.Message
	reactions    *reactions.Tally
	announcement *announcements.Announcement
}

func MessageSent(message messages.Message) Update {
	return Update{message: &message}
}

func ReactionsChanged(tally reactions.Tally) Update {
	return Update{reactions: &tally}
}

func Announced(announcement announcements.Announcement) Update {
	return Update{announcement: &announcement}
}

func (u Update) Message() (messages.Message, bool) {
	if u.message == nil {
		return messages.Message{}, false
	}
	return *u.message, true
}

func (u Update) Reactions() (reactions.Tally, bool) {
	if u.reactions == nil {
		return reactions.Tally{}, false
	}
	return *u.reactions, true
}

func (u Update) Announcement() (announcements.Announcement, bool) {
	if u.announcement == nil {
		return announcements.Announcement{}, false
	}
	return *u.announcement, true
}

func (u Update) MessageID() messages.MessageID {
	if u.reactions != nil {
		return u.reactions.MessageID()
	}
	if u.message != nil {
		return u.message.ID()
	}
	return ""
}
