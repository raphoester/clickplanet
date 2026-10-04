package player_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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

type Accounts interface {
	CreatedAt(ctx context.Context, account players.AccountID) (time.Time, error)
}

func NewPostgresQuery(db cppg.Querier, accounts Accounts, catalog titles.Catalog, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, accounts: accounts, catalog: catalog, clock: clock}
}

type PostgresQuery struct {
	db       cppg.Querier
	accounts Accounts
	catalog  titles.Catalog
	clock    cptime.Clock
}

const player = `
	SELECT
		profiles.account_id,
		profiles.name,
		profiles.admin,
		profiles.color,
		COALESCE(stats.tiles_taken, 0),
		COALESCE(stats.streak_current, 0),
		COALESCE(stats.streak_best, 0),
		stats.streak_last_day,
		COALESCE(worn_titles.title, ''),
		COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = profiles.account_id),
			'{}'
		)
	FROM profiles
	LEFT JOIN stats ON stats.account_id = profiles.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = profiles.account_id
	WHERE profiles.name_folded = $1
`

func (q *PostgresQuery) Player(ctx context.Context, value string) (*playerv1.GetPlayerResponse, error) {
	name, err := players.NameOf(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", players.ErrNoProfile, err)
	}

	var (
		account uuid.UUID
		shown   string
		admin   bool
		color   int32
		stats   players.Stats
		lastDay sql.NullTime
		choice  string
		held    []string
	)
	err = q.db.QueryRowContext(ctx, player, name.Folded()).Scan(
		&account, &shown, &admin, &color,
		&stats.TilesTaken, &stats.StreakCurrent, &stats.StreakBest, &lastDay,
		&choice, pq.Array(&held),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, players.ErrNoProfile
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the player: %w", err)
	}

	kept, err := playermessage.KeptColor(color)
	if err != nil {
		return nil, fmt.Errorf("failed to read the player's color: %w", err)
	}
	stats.Account = players.AccountID(account)
	if lastDay.Valid {
		stats.StreakLastDay = players.DayOf(lastDay.Time)
	}

	createdAt, err := q.accounts.CreatedAt(ctx, stats.Account)
	if err != nil {
		return nil, fmt.Errorf("failed to ask when the account was made: %w", err)
	}

	showcase := wearing.ShowcaseOf(q.catalog, idsOf(held), titles.ID(choice))
	answer := &playerv1.Player{
		Name:      shown,
		Stats:     playermessage.Stats(stats.AsOf(players.DayOf(q.clock.Now()))),
		Admin:     admin,
		Color:     kept,
		Titles:    playermessage.Titles(showcase.Shown),
		WornTitle: playermessage.Title(showcase.Worn),
	}
	if !createdAt.IsZero() {
		answer.CreatedAtUnixMs = createdAt.UnixMilli()
	}
	return &playerv1.GetPlayerResponse{Player: answer}, nil
}

func idsOf(held []string) titles.IDs {
	ids := make(titles.IDs, 0, len(held))
	for _, id := range held {
		ids = append(ids, titles.ID(id))
	}
	return ids
}
