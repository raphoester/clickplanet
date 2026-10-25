package postgres_conviction_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ convictions.Store = (*Store)(nil)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

type reading struct {
	Watchdog string     `json:"watchdog"`
	Level    string     `json:"level"`
	Evidence string     `json:"evidence"`
	At       *time.Time `json:"at"`
}

func (s *Store) Keep(ctx context.Context, conviction convictions.Conviction) error {
	readings := make([]reading, 0, len(conviction.Readings))
	for _, r := range conviction.Readings {
		kept := reading{Watchdog: r.Watchdog, Level: r.Level, Evidence: r.Evidence}
		if !r.At.IsZero() {
			kept.At = &r.At
		}
		readings = append(readings, kept)
	}
	encoded, err := json.Marshal(readings)
	if err != nil {
		return fmt.Errorf("failed to encode the readings of a conviction: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO convictions (convicted_at, scope, account, alongside, flags, offence, banned_until, readings,
			clicks, active_for, longest_gap, top_country, top_country_clicks)
		VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, $6, $7, $8,
			$9, $10 * interval '1 microsecond', $11 * interval '1 microsecond', $12, $13)
	`, conviction.At, conviction.Scope, conviction.Account, conviction.Alongside, conviction.Flags, conviction.Offence,
		conviction.BannedUntil, encoded, conviction.Clicks, conviction.ActiveFor.Microseconds(),
		conviction.LongestGap.Microseconds(), conviction.TopCountry, conviction.TopCountryClicks,
	); err != nil {
		return fmt.Errorf("failed to keep a conviction: %w", err)
	}
	return nil
}

func (s *Store) Between(ctx context.Context, since, until time.Time) ([]convictions.Conviction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT convicted_at, scope, COALESCE(account::text, ''), COALESCE(alongside::text, ''), flags, offence,
			banned_until, readings, clicks, (EXTRACT(EPOCH FROM active_for) * 1000000)::bigint,
			(EXTRACT(EPOCH FROM longest_gap) * 1000000)::bigint, top_country, top_country_clicks
		FROM convictions
		WHERE convicted_at >= $1 AND convicted_at < $2
		ORDER BY convicted_at, id
	`, since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the convictions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var between []convictions.Conviction
	for rows.Next() {
		conviction, err := scan(rows)
		if err != nil {
			return nil, err
		}
		between = append(between, conviction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the convictions: %w", err)
	}

	return between, nil
}

func scan(rows *sql.Rows) (convictions.Conviction, error) {
	var (
		conviction            convictions.Conviction
		encoded               []byte
		activeFor, longestGap int64
	)
	if err := rows.Scan(&conviction.At, &conviction.Scope, &conviction.Account, &conviction.Alongside,
		&conviction.Flags, &conviction.Offence, &conviction.BannedUntil, &encoded, &conviction.Clicks,
		&activeFor, &longestGap, &conviction.TopCountry, &conviction.TopCountryClicks,
	); err != nil {
		return convictions.Conviction{}, fmt.Errorf("failed to read a conviction: %w", err)
	}

	var readings []reading
	if err := json.Unmarshal(encoded, &readings); err != nil {
		return convictions.Conviction{}, fmt.Errorf("failed to decode the readings of a conviction: %w", err)
	}

	conviction.At = conviction.At.UTC()
	conviction.BannedUntil = conviction.BannedUntil.UTC()
	conviction.ActiveFor = time.Duration(activeFor) * time.Microsecond
	conviction.LongestGap = time.Duration(longestGap) * time.Microsecond
	conviction.Readings = make([]detect.Reading, 0, len(readings))
	for _, r := range readings {
		read := detect.Reading{Watchdog: r.Watchdog, Level: r.Level, Evidence: r.Evidence}
		if r.At != nil {
			read.At = r.At.UTC()
		}
		conviction.Readings = append(conviction.Readings, read)
	}

	return conviction, nil
}
