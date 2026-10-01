// Package postgres_allegiance_store keeps the allegiance tallies in planet.allegiances, one row per key.
package postgres_allegiance_store

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ clicks.AllegianceStorage = (*Store)(nil)

// Allegiances reads the tallies under keys. A key with no row is absent.
func (s *Store) Allegiances(ctx context.Context, keys ...clicks.AllegianceKey) (map[clicks.AllegianceKey]clicks.Allegiance, error) {
	names := make([]string, len(keys))
	for i, key := range keys {
		names[i] = string(key)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT key, weights, at FROM allegiances WHERE key = ANY($1::text[])`, pq.Array(names))
	if err != nil {
		return nil, fmt.Errorf("failed to read the allegiances: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tallies := make(map[clicks.AllegianceKey]clicks.Allegiance, len(keys))
	for rows.Next() {
		var (
			key     string
			encoded []byte
			at      time.Time
			weights map[string]float64
		)
		if err := rows.Scan(&key, &encoded, &at); err != nil {
			return nil, fmt.Errorf("failed to scan an allegiance: %w", err)
		}
		if err := json.Unmarshal(encoded, &weights); err != nil {
			return nil, fmt.Errorf("failed to decode the allegiance %q: %w", key, err)
		}
		tallies[clicks.AllegianceKey(key)] = clicks.NewAllegiance(weights, at.UTC())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the allegiances: %w", err)
	}

	return tallies, nil
}

// SaveAllegiances writes every tally, in one statement.
func (s *Store) SaveAllegiances(ctx context.Context, tallies map[clicks.AllegianceKey]clicks.Allegiance) error {
	var keys, weights, ats []string
	for _, key := range slices.Sorted(maps.Keys(tallies)) {
		encoded, err := json.Marshal(tallies[key].Weights())
		if err != nil {
			return fmt.Errorf("failed to encode the allegiance %q: %w", key, err)
		}
		keys = append(keys, string(key))
		weights = append(weights, string(encoded))
		ats = append(ats, tallies[key].At().Format(time.RFC3339Nano))
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO allegiances (key, weights, at)
		SELECT key, weights::jsonb, at::timestamptz FROM unnest($1::text[], $2::text[], $3::text[]) AS t(key, weights, at)
		ON CONFLICT (key) DO UPDATE SET weights = EXCLUDED.weights, at = EXCLUDED.at
	`, pq.Array(keys), pq.Array(weights), pq.Array(ats)); err != nil {
		return fmt.Errorf("failed to write the allegiances: %w", err)
	}

	return nil
}

// DeleteAllegiancesBefore deletes the tallies whose last take is before cutoff, and says how many went.
func (s *Store) DeleteAllegiancesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM allegiances WHERE at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete the faded allegiances: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the faded allegiances: %w", err)
	}

	return deleted, nil
}
