package accounts

import "github.com/google/uuid"

// IDProvider gives each new account its id.
type IDProvider interface {
	NewID() (uuid.UUID, error)
}

// TokenGenerator gives each new session its secret.
type TokenGenerator interface {
	NewToken() (*Token, error)
}
