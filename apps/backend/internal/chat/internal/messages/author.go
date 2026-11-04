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

// Who posts, as the chat shows it: its badges, a streak and a title, are not the chat's, and the page reads them by
// the message's account.
type Author struct {
	name  string
	admin bool
	color int32
	guest bool
}

func AuthorOf(name string, admin bool, color int32) Author {
	return Author{name: name, admin: admin, color: color}
}

// A guest holds no color.
func GuestOf(name string) Author {
	return Author{name: name, guest: true}
}

func (a Author) Name() string { return a.name }

func (a Author) Admin() bool { return a.admin }

func (a Author) Color() int32 { return a.color }

var ErrAuthorUnavailable = errors.New("the sender could not be identified")
