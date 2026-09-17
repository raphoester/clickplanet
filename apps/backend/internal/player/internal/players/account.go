// Package players is what the game keeps about one player: the name it chose, and what it did on the map.
//
// A player is an account, the one the click token names. This module never makes an account: auth does,
// and planet tells it of every tile one took.
package players

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// AccountID names a player. It is the click token's own type, so an account read off a token needs no conversion.
type AccountID = cpsession.AccountID

var ErrInvalidAccount = errors.New("not an account id")

// AccountIDOf reads an account id off the wire or an event. The nil UUID is no account, and refused.
func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}
