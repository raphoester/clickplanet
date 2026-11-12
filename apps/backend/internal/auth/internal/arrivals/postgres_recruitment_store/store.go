package postgres_recruitment_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ arrivals.Recruitments = (*Store)(nil)

func (s *Store) Save(ctx context.Context, saved ...arrivals.Recruitment) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, recruitment := range saved {
			kept := recruitment.TakeSnapshot()
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO recruitments (arrival_id, recruiter_id)
				VALUES ($1, $2)
				ON CONFLICT (arrival_id) DO UPDATE SET recruiter_id = EXCLUDED.recruiter_id
			`, uuid.UUID(kept.Arrival), recruiter(kept.Recruiter)); err != nil {
				return fmt.Errorf("failed to upsert a recruitment: %w", err)
			}
		}
		return nil
	})
}

// An anonymized recruitment has no recruiter: NULL.
func recruiter(account arrivals.AccountID) uuid.NullUUID {
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: account != arrivals.AccountID{}}
}

func (s *Store) By(ctx context.Context, account arrivals.AccountID) ([]arrivals.Recruitment, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT arrival_id, recruiter_id
		FROM recruitments
		WHERE recruiter_id = $1
		ORDER BY arrival_id
	`, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read the recruitments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := []arrivals.Recruitment{}
	for rows.Next() {
		var (
			arrival uuid.UUID
			held    uuid.NullUUID
		)
		if err := rows.Scan(&arrival, &held); err != nil {
			return nil, fmt.Errorf("failed to read a recruitment: %w", err)
		}
		recruitment, err := arrivals.RecruitmentSnapshot{
			Arrival: arrivals.ArrivalID(arrival), Recruiter: arrivals.AccountID(held.UUID),
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the recruitment of %s: %w", arrival, err)
		}
		found = append(found, recruitment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the recruitments: %w", err)
	}
	return found, nil
}
