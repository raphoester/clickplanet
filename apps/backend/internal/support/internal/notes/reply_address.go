package notes

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

type AddressBook interface {
	// ErrNoReplyAddress for an account that holds none.
	Address(ctx context.Context, account AccountID) (cpmail.Address, error)
}

// Where a note is answered: the address its player gave, or else the one its account holds.
func ReplyAddress(ctx context.Context, given cpmail.Address, account AccountID, book AddressBook) (cpmail.Address, error) {
	if given != "" {
		return given, nil
	}
	return book.Address(ctx, account) //nolint:wrapcheck // the caller reads ErrNoReplyAddress.
}
