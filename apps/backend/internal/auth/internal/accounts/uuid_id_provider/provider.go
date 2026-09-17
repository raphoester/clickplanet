// Package uuid_id_provider gives accounts time-ordered UUIDv7 ids, so the table's primary key grows in insert order.
package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Provider struct{}

var _ accounts.IDProvider = Provider{}

func (Provider) NewID() (accounts.AccountID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return accounts.AccountID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return accounts.AccountID(id), nil
}
