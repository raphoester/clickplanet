package standings

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Store interface {
	// RecordTakes counts an account's takes in a book, in ledger order, in one write: a take told again changes nothing.
	RecordTakes(ctx context.Context, book books.BookID, account AccountID, takes []Take) error
	DeleteAccount(ctx context.Context, account AccountID) error
}
