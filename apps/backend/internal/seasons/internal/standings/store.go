package standings

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type Store interface {
	RecordTake(ctx context.Context, season calendar.Number, take Take) error
	DeleteAccount(ctx context.Context, account AccountID) error
}
