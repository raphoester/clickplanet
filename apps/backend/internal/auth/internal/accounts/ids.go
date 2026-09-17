package accounts

import "github.com/google/uuid"

// AccountID names an account. A type of its own, so no other id compiles where an account is asked for.
type AccountID uuid.UUID

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}

// TokenHash names a session: the SHA-256 of its cookie's token, the only form of it that is stored.
type TokenHash []byte
