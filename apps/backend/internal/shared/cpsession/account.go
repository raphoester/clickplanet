package cpsession

import (
	"time"

	"github.com/google/uuid"
)

// AccountID names the account a click token is minted for. It lives with the token: auth mints it and planet reads it.
type AccountID uuid.UUID

// NoAccount is the account of a token minted for nobody.
var NoAccount = AccountID{}

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}

// CreatedAt is when the account was made, read off the UUIDv7 auth makes; false for any other id.
func (id AccountID) CreatedAt() (time.Time, bool) {
	u := uuid.UUID(id)
	if u.Version() != 7 || u.Variant() != uuid.RFC4122 {
		return time.Time{}, false
	}

	var ms int64
	for _, b := range u[:6] {
		ms = ms<<8 | int64(b)
	}

	return time.UnixMilli(ms).UTC(), true
}
