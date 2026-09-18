package messages

import (
	"errors"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// AccountID names the sender's account. It is the click token's own type, so an account read off a token needs
// no conversion.
type AccountID = cpsession.AccountID

// AccountIDOf reads the account the session interceptor put on the context. Anything that is not an account,
// the empty string of a caller with no token included, is cpsession.NoAccount.
func AccountIDOf(value string) AccountID {
	id, err := uuid.Parse(value)
	if err != nil {
		return cpsession.NoAccount
	}
	return AccountID(id)
}

// ErrNoAccount is a caller whose token names no account, or who sent none. Only an account posts or reacts:
// its name is the one the player module gives it, so nobody chooses a guest's name.
var ErrNoAccount = errors.New("only an account may post or react")

// Author is who posts, as the player module answers it: the account's username, or "guest_" and its guest code.
type Author struct {
	Name string
	// Admin is false for a guest.
	Admin bool
}

// ErrAuthorUnavailable is a sender the player module could not name. The message is refused: it would have no
// name.
var ErrAuthorUnavailable = errors.New("the sender could not be identified")
