package accounts

import "context"

// Sessions keeps accounts and their sessions; an unknown token hash is ErrSessionNotFound. SessionsContractSuite is its behaviour.
type Sessions interface {
	FindSession(ctx context.Context, tokenHash []byte) (*Session, error)
	CreateGuest(ctx context.Context, session *Session) error
	SaveSession(ctx context.Context, session *Session) error
}
