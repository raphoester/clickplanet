package postgres_judgement_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ judgements.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, judgement judgements.Judgement) error {
	kept := judgement.TakeSnapshot()
	var message sql.NullString
	if kept.Message != judgements.NoMessage {
		message = sql.NullString{String: string(kept.Message), Valid: true}
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO judgements (report_id, message_id, verdict, judged, flagged, reason, judged_at)
		VALUES ($1, $2, $3::judgement_verdict, $4, $5, $6, $7)
		ON CONFLICT (report_id) DO NOTHING
	`, uuid.UUID(kept.Report), message, kept.Verdict.String(), kept.Judged, kept.Flagged, kept.Reason, kept.At.UTC()); err != nil {
		return fmt.Errorf("failed to keep a judgement: %w", err)
	}
	return nil
}

const columns = `report_id, COALESCE(message_id, ''), verdict::text, judged, flagged, reason, judged_at`

func (s *Store) OfReport(ctx context.Context, report judgements.ReportID) (judgements.Judgement, error) {
	return s.one(ctx, `SELECT `+columns+` FROM judgements WHERE report_id = $1`, uuid.UUID(report))
}

func (s *Store) OfMessage(ctx context.Context, message messages.MessageID) (judgements.Judgement, error) {
	return s.one(ctx, `
		SELECT `+columns+` FROM judgements WHERE message_id = $1 AND judged ORDER BY judged_at, report_id LIMIT 1
	`, string(message))
}

func (s *Store) one(ctx context.Context, query string, arg any) (judgements.Judgement, error) {
	var (
		report  uuid.UUID
		snap    judgements.Snapshot
		message string
		verdict string
	)
	err := s.db.On(ctx).QueryRowContext(ctx, query, arg).
		Scan(&report, &message, &verdict, &snap.Judged, &snap.Flagged, &snap.Reason, &snap.At)
	if errors.Is(err, sql.ErrNoRows) {
		return judgements.Judgement{}, judgements.ErrNotJudged
	}
	if err != nil {
		return judgements.Judgement{}, fmt.Errorf("failed to read a judgement: %w", err)
	}
	snap.Report, snap.Message, snap.At = judgements.ReportID(report), messages.MessageID(message), snap.At.UTC()
	if snap.Verdict, err = judgements.VerdictOf(verdict); err != nil {
		return judgements.Judgement{}, err //nolint:wrapcheck // the sentinel names it.
	}
	judgement, err := snap.Restore()
	if err != nil {
		return judgements.Judgement{}, fmt.Errorf("failed to restore a judgement: %w", err)
	}
	return judgement, nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM judgements WHERE judged_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune the judgements: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the judgements pruned: %w", err)
	}
	return deleted, nil
}
