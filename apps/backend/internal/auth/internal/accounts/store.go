package accounts

import (
	"context"
	"time"
)

type Store interface {
	Session(ctx context.Context, tokenHash TokenHash) (*Session, error)
	CreateGuest(ctx context.Context, session *Session) error
	SaveSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, tokenHash TokenHash) error
	DeleteSessions(ctx context.Context, account AccountID) error

	Account(ctx context.Context, account AccountID) (*Account, error)
	Identity(ctx context.Context, provider string, subject string) (*Identity, error)
	AccountOfEmail(ctx context.Context, address string) (*Account, error)
	SaveSignIn(ctx context.Context, signIn SignIn) error
	RequestDeletion(ctx context.Context, account AccountID, at time.Time) error
	DeleteAccount(ctx context.Context, account AccountID) error
	PruneGuests(ctx context.Context, idleSince time.Time, limit int) ([]AccountID, error)
	PurgeDeletions(ctx context.Context, requestedBefore time.Time, limit int) ([]AccountID, error)
}
