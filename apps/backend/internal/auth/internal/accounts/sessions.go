package accounts

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Sessions is where accounts and their sessions are kept. SessionsContractSuite is its behaviour.
type Sessions interface {
	FindSession(ctx context.Context, tokenHash []byte) (Session, bool, error)
	ExtendSession(ctx context.Context, tokenHash []byte, expiresAt, now time.Time) error
	CreateGuest(ctx context.Context, account uuid.UUID, tokenHash []byte, expiresAt, now time.Time) error
}
