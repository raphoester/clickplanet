package feed

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

type Update struct {
	message      *messages.Message
	reactions    *reactions.Tally
	reactedTo    *messages.Message
	announcement *announcements.Announcement
	removed      *messages.MessageID
	left         *departure
}

type departure struct {
	account messages.AccountID
	room    messages.Room
}

func MessageSent(message messages.Message) Update {
	return Update{message: &message}
}

func ReactionsChanged(reactedTo messages.Message, tally reactions.Tally) Update {
	return Update{reactions: &tally, reactedTo: &reactedTo}
}

func Announced(announcement announcements.Announcement) Update {
	return Update{announcement: &announcement}
}

func MessageRemoved(message messages.MessageID) Update {
	return Update{removed: &message}
}

func RoomLeft(account messages.AccountID, room messages.Room) Update {
	return Update{left: &departure{account: account, room: room}}
}

// A removal goes to every room: it carries an id alone, which a client that does not hold it drops.
func (u Update) For(room messages.Room) bool {
	switch {
	case u.message != nil:
		return u.message.In(room)
	case u.reactions != nil:
		return u.reactedTo.In(room)
	case u.announcement != nil:
		return room == messages.World
	case u.removed != nil:
		return true
	case u.left != nil:
		return false
	}
	return false
}

func (u Update) Leaves(account messages.AccountID, room messages.Room) bool {
	return u.left != nil && u.left.account == account && u.left.room == room
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

func (u Update) Removed() (messages.MessageID, bool) {
	if u.removed == nil {
		return "", false
	}
	return *u.removed, true
}
