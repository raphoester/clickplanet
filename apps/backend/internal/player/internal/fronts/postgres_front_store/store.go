package postgres_front_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ fronts.Store = (*Store)(nil)

// In country order, so two takes crossing the same two countries lock their rows in one order.
const recordTake = `
	INSERT INTO fronts AS kept (account_id, country, plays_for, plays_against)
	SELECT $1, country, plays_for, plays_against
	FROM (VALUES ($2::text, 1::bigint, 0::bigint), ($3::text, 0::bigint, 1::bigint)) AS take (country, plays_for, plays_against)
	WHERE country <> ''
	ORDER BY country
	ON CONFLICT (account_id, country) DO UPDATE SET
		plays_for = kept.plays_for + excluded.plays_for,
		plays_against = kept.plays_against + excluded.plays_against
`

func (s *Store) RecordTake(ctx context.Context, take fronts.Take) error {
	var against fronts.Country
	if country, ok := take.Against(); ok {
		against = country
	}

	if _, err := s.db.ExecContext(ctx, recordTake,
		uuid.UUID(take.Account()), string(take.Country()), string(against)); err != nil {
		return fmt.Errorf("failed to count the take for and against its countries: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM fronts WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's fronts: %w", err)
	}
	return nil
}
