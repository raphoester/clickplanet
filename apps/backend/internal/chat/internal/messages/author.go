package messages

import (
	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// AccountID names the sender's account. It is the click token's own type, so an account read off a token needs
// no conversion.
type AccountID = cpsession.AccountID

// AccountIDOf reads the account the session interceptor put on the context. Anything that is not an account,
// the empty string of a caller with no token included, is cpsession.NoAccount: a guest.
func AccountIDOf(value string) AccountID {
	id, err := uuid.Parse(value)
	if err != nil {
		return cpsession.NoAccount
	}
	return AccountID(id)
}

// GuestPrefix goes before the name every guest types. No username starts with it, so a guest cannot pass for
// a player.
const GuestPrefix = "guest_"

// GuestName is the name a guest typed, cleaned by Limits.Name and then prefixed: the bound is on what the guest
// typed, not on the prefix.
func (l Limits) GuestName(value string) (string, error) {
	name, err := l.Name(value)
	if err != nil {
		return "", err
	}
	return GuestPrefix + name, nil
}
