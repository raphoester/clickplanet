//go:build testing

package postgres_contribution_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

func (s *Store) RecordTake(ctx context.Context, season calendar.Number, take standings.Take) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin counting the take: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	key := seasonAccount{season: season, account: take.Account}
	tally, err := tallyOf(ctx, tx, season, take.Account)
	if err != nil {
		return err
	}
	if err := saveTally(ctx, tx, key, tally.WithTake(take.Country)); err != nil {
		return err
	}
	return tx.Commit() //nolint:wrapcheck // a test helper.
}
