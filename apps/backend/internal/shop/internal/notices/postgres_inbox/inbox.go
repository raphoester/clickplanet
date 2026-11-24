package postgres_inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
)

func New(db cppg.Conn) *Inbox {
	return &Inbox{db: db}
}

type Inbox struct {
	db cppg.Conn
}

var _ notices.Inbox = (*Inbox)(nil)

func (i *Inbox) Keep(ctx context.Context, id notices.EventID, at time.Time) (bool, error) {
	result, err := i.db.On(ctx).ExecContext(ctx,
		`INSERT INTO stripe_events (id, received_at) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, id.String(), at.UTC())
	if err != nil {
		return false, fmt.Errorf("failed to keep the event %s: %w", id, err)
	}
	kept, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to count the event %s kept: %w", id, err)
	}
	return kept == 1, nil
}
