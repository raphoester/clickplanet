package postgres_player_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ players.Store = (*Store)(nil)

func (s *Store) Profile(ctx context.Context, account players.AccountID) (players.Profile, error) {
	var (
		name      string
		updatedAt time.Time
		chosenAt  sql.NullTime
		admin     bool
		color     int32
	)
	err := s.db.On(ctx).QueryRowContext(
		ctx,
		`SELECT name, updated_at, name_chosen_at, admin, color FROM profiles WHERE account_id = $1`, uuid.UUID(account),
	).Scan(&name, &updatedAt, &chosenAt, &admin, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Profile{}, players.ErrNoProfile
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	snapshot := players.ProfileSnapshot{
		Account: account, Name: players.Name(name), UpdatedAt: updatedAt.UTC(), Admin: admin, Color: players.Color(color),
	}
	if chosenAt.Valid {
		snapshot.ChosenAt = chosenAt.Time.UTC()
	}
	return snapshot.Restore(), nil
}

const uniqueNameIndex = "profiles_name_key"

const uniqueViolation = "23505"

func (s *Store) SaveProfile(ctx context.Context, profile players.Profile) error {
	kept := profile.TakeSnapshot()
	_, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, name_folded, updated_at, name_chosen_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id) DO UPDATE SET
			name = excluded.name, name_folded = excluded.name_folded, updated_at = excluded.updated_at,
			name_chosen_at = excluded.name_chosen_at
	`, uuid.UUID(kept.Account), string(kept.Name), kept.Name.Folded(), kept.UpdatedAt.UTC(), nullableTime(kept.ChosenAt))

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueNameIndex {
		return players.ErrNameTaken
	}
	if err != nil {
		return fmt.Errorf("failed to save the profile: %w", err)
	}
	return nil
}

// A taken name raises no error, so the transaction it runs in, which records the player made, goes on to the next draw.
func (s *Store) CreateProfile(ctx context.Context, profile players.Profile) error {
	kept := profile.TakeSnapshot()
	result, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO profiles (account_id, name, name_folded, updated_at, name_chosen_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`, uuid.UUID(kept.Account), string(kept.Name), kept.Name.Folded(), kept.UpdatedAt.UTC(), nullableTime(kept.ChosenAt))
	if err != nil {
		return fmt.Errorf("failed to create the profile: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count the profiles created: %w", err)
	}
	if created == 1 {
		return nil
	}

	var exists bool
	if err := s.db.On(ctx).QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE account_id = $1)`,
		uuid.UUID(kept.Account)).Scan(&exists); err != nil {
		return fmt.Errorf("failed to read whether the account has a profile: %w", err)
	}
	if exists {
		return players.ErrProfileExists
	}
	return players.ErrNameTaken
}

func (s *Store) SaveColor(ctx context.Context, account players.AccountID, color players.Color) error {
	result, err := s.db.On(ctx).ExecContext(ctx, `UPDATE profiles SET color = $2 WHERE account_id = $1`, uuid.UUID(account), int32(color))
	if err != nil {
		return fmt.Errorf("failed to save the color: %w", err)
	}
	saved, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count the colors saved: %w", err)
	}
	if saved == 0 {
		return players.ErrNoProfile
	}
	return nil
}

func (s *Store) GuestCode(ctx context.Context, account players.AccountID) (players.GuestCode, error) {
	var code string
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT code FROM guest_codes WHERE account_id = $1`, uuid.UUID(account)).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", players.ErrNoGuestCode
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the guest code: %w", err)
	}
	return players.GuestCode(code), nil
}

const uniqueGuestCode = "guest_codes_code_key"

func (s *Store) SaveGuestCode(ctx context.Context, account players.AccountID, code players.GuestCode) error {
	_, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO guest_codes (account_id, code) VALUES ($1, $2)
		ON CONFLICT (account_id) DO NOTHING
	`, uuid.UUID(account), string(code))

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueGuestCode {
		return players.ErrGuestCodeTaken
	}
	if err != nil {
		return fmt.Errorf("failed to save the guest code: %w", err)
	}
	return nil
}

const statsColumns = `account_id, tiles_taken, streak_current, streak_best, streak_last_day, messages_sent,
	bombs_dropped, tiles_fortified, bonuses_claimed, right_answers, missions_done, counted_takes_below, counted_bombs_below,
	counted_fortifies_below, counted_messages_until, counted_bonuses_until, counted_answers_until, counted_takes_below_index`

func (s *Store) Stats(ctx context.Context, account players.AccountID) (players.Stats, error) {
	return statsOf(s.db.On(ctx).QueryRowContext(ctx, `SELECT `+statsColumns+` FROM stats WHERE account_id = $1`, uuid.UUID(account)))
}

const statsLock = 0x706c6179

func (s *Store) Record(ctx context.Context, account players.AccountID, change func(players.Stats) players.Stats) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, statsLock, account.String()); err != nil {
			return fmt.Errorf("failed to lock the account's stats: %w", err)
		}

		current, err := statsOf(tx.QueryRowContext(ctx, `SELECT `+statsColumns+` FROM stats WHERE account_id = $1`, uuid.UUID(account)))
		if errors.Is(err, players.ErrNoStats) {
			current = players.NewStats(account)
			err = nil
		}
		if err != nil {
			return err
		}

		next := change(current).TakeSnapshot()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO stats (`+statsColumns+`)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
			ON CONFLICT (account_id) DO UPDATE SET
				tiles_taken = excluded.tiles_taken,
				streak_current = excluded.streak_current,
				streak_best = excluded.streak_best,
				streak_last_day = excluded.streak_last_day,
				messages_sent = excluded.messages_sent,
				bombs_dropped = excluded.bombs_dropped,
				tiles_fortified = excluded.tiles_fortified,
				bonuses_claimed = excluded.bonuses_claimed,
				right_answers = excluded.right_answers,
				missions_done = excluded.missions_done,
				counted_takes_below = excluded.counted_takes_below,
				counted_bombs_below = excluded.counted_bombs_below,
				counted_fortifies_below = excluded.counted_fortifies_below,
				counted_messages_until = excluded.counted_messages_until,
				counted_bonuses_until = excluded.counted_bonuses_until,
				counted_answers_until = excluded.counted_answers_until,
				counted_takes_below_index = excluded.counted_takes_below_index
		`, uuid.UUID(account), int64(next.TilesTaken), //nolint:gosec // counted an act at a time: never past int64.
			int64(next.Streak.Days()), int64(next.StreakBest), nullableDay(next.Streak.LastDay()),
			int64(next.MessagesSent), int64(next.BombsDropped), int64(next.TilesFortified), //nolint:gosec // as above.
			int64(next.BonusesClaimed), int64(next.RightAnswers), int64(next.MissionsDone), //nolint:gosec // as above.
			int64(next.Counted.TakesBelow.Position), int64(next.Counted.BombsBelow), int64(next.Counted.FortifiesBelow), //nolint:gosec // ledger positions fit in int64.
			nullableTime(next.Counted.MessagesUntil), nullableTime(next.Counted.BonusesUntil),
			nullableTime(next.Counted.AnswersUntil), int64(next.Counted.TakesBelow.Index)); err != nil {
			return fmt.Errorf("failed to save the stats: %w", err)
		}
		return nil
	})
}

func (s *Store) StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT `+statsColumns+` FROM stats
		WHERE account_id > $1 ORDER BY account_id LIMIT $2
	`, uuid.UUID(after), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read a page of stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var page []players.Stats
	for rows.Next() {
		stats, err := statsOf(rows)
		if err != nil {
			return nil, err
		}
		page = append(page, stats)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read a page of stats: %w", err)
	}
	return page, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, statement := range []string{
			`DELETE FROM profiles WHERE account_id = $1`,
			`DELETE FROM guest_codes WHERE account_id = $1`,
			`DELETE FROM stats WHERE account_id = $1`,
		} {
			if _, err := tx.ExecContext(ctx, statement, uuid.UUID(account)); err != nil {
				return fmt.Errorf("failed to delete the account's rows: %w", err)
			}
		}
		return nil
	})
}

type scanner interface {
	Scan(dest ...any) error
}

func statsOf(row scanner) (players.Stats, error) {
	var (
		account                                                                    uuid.UUID
		tiles, current, best, messages, bombs, fortified, claimed, right, missions int64
		takesPosition, takesIndex, bombsBelow, fortifiesBelow                      int64
		lastDay, messagesUntil, bonusesUntil, answersUntil                         sql.NullTime
	)
	err := row.Scan(&account, &tiles, &current, &best, &lastDay, &messages, &bombs, &fortified, &claimed, &right, &missions,
		&takesPosition, &bombsBelow, &fortifiesBelow, &messagesUntil, &bonusesUntil, &answersUntil, &takesIndex)
	if errors.Is(err, sql.ErrNoRows) {
		return players.Stats{}, players.ErrNoStats
	}
	if err != nil {
		return players.Stats{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	takesBelow := players.TakeMark{Position: uint64(takesPosition), Index: uint32(takesIndex)} //nolint:gosec // every mark has a CHECK (>= 0).
	var day players.Day
	if lastDay.Valid {
		day = players.DayOf(lastDay.Time)
	}
	return players.StatsSnapshot{
		Account:        players.AccountID(account),
		TilesTaken:     uint64(tiles),                          //nolint:gosec // every counter has a CHECK (>= 0).
		Streak:         players.StreakOf(uint32(current), day), //nolint:gosec // as above, and one a day.
		StreakBest:     uint32(best),                           //nolint:gosec // as above.
		MessagesSent:   uint64(messages),                       //nolint:gosec // as above.
		BombsDropped:   uint64(bombs),                          //nolint:gosec // as above.
		TilesFortified: uint64(fortified),                      //nolint:gosec // as above.
		BonusesClaimed: uint64(claimed),                        //nolint:gosec // as above.
		RightAnswers:   uint64(right),                          //nolint:gosec // as above.
		MissionsDone:   uint64(missions),                       //nolint:gosec // as above.
		Counted: players.Counted{
			TakesBelow:     takesBelow,
			BombsBelow:     uint64(bombsBelow),     //nolint:gosec // as above.
			FortifiesBelow: uint64(fortifiesBelow), //nolint:gosec // as above.
			MessagesUntil:  timeOf(messagesUntil),
			BonusesUntil:   timeOf(bonusesUntil),
			AnswersUntil:   timeOf(answersUntil),
		},
	}.Restore(), nil
}

func timeOf(at sql.NullTime) time.Time {
	if !at.Valid {
		return time.Time{}
	}
	return at.Time.UTC()
}

func nullableDay(day players.Day) sql.NullString {
	return sql.NullString{String: day.String(), Valid: !day.Empty()}
}

func nullableTime(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at.UTC(), Valid: !at.IsZero()}
}
