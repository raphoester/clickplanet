package postgres_notice_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ notices.Store = (*Store)(nil)

func (s *Store) Noticed(ctx context.Context, account string) (notices.Notice, bool, error) {
	var (
		notice       notices.Notice
		acknowledged sql.NullTime
	)
	err := s.db.On(ctx).QueryRowContext(ctx,
		`SELECT account::text, alongside::text, noticed_at, acknowledged_at FROM notices WHERE account = $1`, account).
		Scan(&notice.Account, &notice.Alongside, &notice.At, &acknowledged)
	if errors.Is(err, sql.ErrNoRows) {
		return notices.Notice{}, false, nil
	}
	if err != nil {
		return notices.Notice{}, false, fmt.Errorf("failed to read the notice: %w", err)
	}
	notice.At = notice.At.UTC()
	if acknowledged.Valid {
		notice.AcknowledgedAt = acknowledged.Time.UTC()
	}
	return notice, true, nil
}

func (s *Store) Keep(ctx context.Context, notice notices.Notice) (bool, error) {
	result, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO notices (account, alongside, noticed_at) VALUES ($1, $2, $3) ON CONFLICT (account) DO NOTHING`,
		notice.Account, notice.Alongside, notice.At.UTC())
	if err != nil {
		return false, fmt.Errorf("failed to keep the notice: %w", err)
	}
	kept, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to keep the notice: %w", err)
	}
	return kept == 1, nil
}

func (s *Store) Acknowledge(ctx context.Context, account string, at time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx,
		`UPDATE notices SET acknowledged_at = $2 WHERE account = $1 AND acknowledged_at IS NULL`,
		account, at.UTC()); err != nil {
		return fmt.Errorf("failed to keep the acknowledgement: %w", err)
	}
	return nil
}
