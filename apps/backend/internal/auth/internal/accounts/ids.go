package accounts

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// AccountID names an account. It is the click token's own type, so an account reaches the token with no conversion.
type AccountID = cpsession.AccountID

// AccountIDOf reads an account id another module sent. The nil UUID is no account, and refused.
func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

// TokenHash names a session: the SHA-256 of its cookie's token, the only form of it that is stored.
type TokenHash []byte
