//go:build testing

package notes

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

type FixedAddressBook map[AccountID]cpmail.Address

func (b FixedAddressBook) Address(_ context.Context, account AccountID) (cpmail.Address, error) {
	address, held := b[account]
	if !held {
		return "", fmt.Errorf("%w: %s holds none", ErrNoReplyAddress, account)
	}
	return address, nil
}
