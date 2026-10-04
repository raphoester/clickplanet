package authors_query

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Querier, catalog titles.Catalog, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, catalog: catalog, clock: clock}
}

type PostgresQuery struct {
	db      cppg.Querier
	catalog titles.Catalog
	clock   cptime.Clock
}

const authors = `
	SELECT
		asked.account_id,
		profiles.account_id IS NULL,
		COALESCE(profiles.name, ''),
		COALESCE(profiles.admin, false),
		COALESCE(profiles.color, 0),
		COALESCE(guest_codes.code, ''),
		COALESCE(stats.streak_current, 0),
		stats.streak_last_day,
		COALESCE(worn_titles.title, ''),
		COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = asked.account_id),
			'{}'
		)
	FROM (
		SELECT account_id, min(position) AS position
		FROM unnest($1::uuid[]) WITH ORDINALITY AS asked (account_id, position)
		GROUP BY account_id
	) asked
	LEFT JOIN profiles ON profiles.account_id = asked.account_id
	LEFT JOIN guest_codes ON guest_codes.account_id = asked.account_id
	LEFT JOIN stats ON stats.account_id = asked.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = asked.account_id
	WHERE profiles.account_id IS NOT NULL OR guest_codes.account_id IS NOT NULL
	ORDER BY asked.position
`

func (q *PostgresQuery) Authors(ctx context.Context, accounts []players.AccountID) (*playerv1.GetAuthorsResponse, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	rows, err := q.db.QueryContext(ctx, authors, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the authors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	today := players.DayOf(q.clock.Now())
	answer := &playerv1.GetAuthorsResponse{Authors: make([]*playerv1.Author, 0, len(accounts))}
	for rows.Next() {
		author, err := q.authorOf(rows, today)
		if err != nil {
			return nil, err
		}
		answer.Authors = append(answer.Authors, author)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the authors: %w", err)
	}
	return answer, nil
}

func (q *PostgresQuery) authorOf(rows *sql.Rows, today players.Day) (*playerv1.Author, error) {
	var (
		account uuid.UUID
		guest   bool
		name    string
		admin   bool
		color   int32
		code    string
		streak  uint32
		lastDay sql.NullTime
		choice  string
		owned   []string
	)
	if err := rows.Scan(&account, &guest, &name, &admin, &color, &code, &streak, &lastDay, &choice, pq.Array(&owned)); err != nil {
		return nil, fmt.Errorf("failed to scan an author: %w", err)
	}
	kept, err := playermessage.KeptColor(color)
	if err != nil {
		return nil, fmt.Errorf("failed to read the color of %s: %w", account, err)
	}

	held := make(titles.IDs, 0, len(owned))
	for _, id := range owned {
		held = append(held, titles.ID(id))
	}
	run := players.Streak{Days: streak}
	if lastDay.Valid {
		run.LastDay = players.DayOf(lastDay.Time)
	}

	shown := wearing.AuthorOf(players.Author{
		Name:   players.DisplayNameOf(players.Name(name), players.GuestCode(code)),
		Guest:  guest,
		Admin:  admin,
		Streak: run,
	}.Shown(today), wearing.ShowcaseOf(q.catalog, held, titles.ID(choice)).Worn)

	return &playerv1.Author{
		AccountId: account.String(),
		Name:      shown.Name,
		Admin:     shown.Admin,
		Color:     kept,
		Streak:    shown.Streak.Days,
		WornTitle: playermessage.Title(shown.Worn),
	}, nil
}
