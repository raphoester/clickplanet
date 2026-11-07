package messages

import (
	"errors"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var NoAccount = cpsession.NoAccount

func AccountIDOf(value string) AccountID {
	id, err := uuid.Parse(value)
	if err != nil {
		return cpsession.NoAccount
	}
	return AccountID(id)
}

var ErrNoAccount = errors.New("only an account may post or react")

// Who posts, as the chat's rules see it. A message never carries it: the page names an author by the message's account.
type Author struct {
	name  string
	guest bool
	admin bool
}

func AuthorOf(name string) Author {
	return Author{name: name}
}

func GuestOf(name string) Author {
	return Author{name: name, guest: true}
}

// An admin of the game: the operator's own account, which the chat's judge never acts on.
func AdminOf(name string) Author {
	return Author{name: name, admin: true}
}

// The name a mute announcement keeps.
func (a Author) Name() string { return a.name }

// A guest posts no GIF, nor anything while raid mode runs.
func (a Author) Guest() bool { return a.guest }

func (a Author) Admin() bool { return a.admin }

var ErrAuthorUnavailable = errors.New("the sender could not be identified")
