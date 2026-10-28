package standings

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Store interface {
	RecordTake(ctx context.Context, book books.BookID, take Take) error
	DeleteAccount(ctx context.Context, account AccountID) error
}
