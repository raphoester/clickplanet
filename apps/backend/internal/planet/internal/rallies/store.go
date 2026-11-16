package rallies

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Store keeps each account's last rally, on or ended: none for an account that never called one.
type Store interface {
	Last(ctx context.Context, account cpsession.AccountID) (Rally, error)
	// Change runs under a lock on the account, and keeps what it answers.
	Change(ctx context.Context, account cpsession.AccountID, change func(Rally) (Rally, error)) error
	// On is every rally of the flag that ends after at.
	On(ctx context.Context, flag string, at time.Time) ([]Rally, error)
}

// Teller tells a rally called, once it is kept.
type Teller interface {
	Called(rally Rally)
}

// Tellers tells every rally to each of them.
type Tellers []Teller

func (t Tellers) Called(rally Rally) {
	for _, each := range t {
		each.Called(rally)
	}
}
