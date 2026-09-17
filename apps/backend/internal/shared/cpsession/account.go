package cpsession

import "github.com/google/uuid"

// AccountID names the account a click token is minted for. It lives with the token: auth mints it and planet reads it.
type AccountID uuid.UUID

// NoAccount is the account of a token minted for nobody.
var NoAccount = AccountID{}

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}
