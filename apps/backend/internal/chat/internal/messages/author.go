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

type Author struct {
	Name  string
	Admin bool
}

var ErrAuthorUnavailable = errors.New("the sender could not be identified")
