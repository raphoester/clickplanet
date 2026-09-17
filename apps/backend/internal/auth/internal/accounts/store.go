package accounts

import (
	"context"
	"time"
)

// Store keeps accounts, their identities and their sessions. StoreContractSuite is its behaviour.
type Store interface {
	// FindSession answers ErrSessionNotFound for an unknown token hash.
	FindSession(ctx context.Context, tokenHash TokenHash) (*Session, error)
	CreateGuest(ctx context.Context, session *Session) error
	// SaveSession writes the session's expiry and marks its account seen; ErrSessionNotFound when it is gone.
	SaveSession(ctx context.Context, session *Session) error
	// DeleteSession ends one session. An unknown token hash is not an error.
	DeleteSession(ctx context.Context, tokenHash TokenHash) error
	DeleteSessions(ctx context.Context, account AccountID) error

	// FindAccount answers ErrAccountNotFound for an unknown id.
	FindAccount(ctx context.Context, account AccountID) (*Account, error)
	// FindIdentity answers ErrIdentityNotFound when no account has it.
	FindIdentity(ctx context.Context, provider string, subject string) (*Identity, error)
	// SaveSignIn writes it whole or not at all, and marks the account seen; ErrIdentityTaken when another sign-in linked the identity first.
	SaveSignIn(ctx context.Context, signIn SignIn) error
	// DeleteAccount deletes the account, its identities and its sessions. An unknown id is not an error.
	DeleteAccount(ctx context.Context, account AccountID) error
	// PruneGuests deletes at most limit accounts with no identity, last seen before idleSince, and says how many.
	PruneGuests(ctx context.Context, idleSince time.Time, limit int) (int, error)
}
