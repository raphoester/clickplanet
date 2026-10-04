package player_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Titles interface {
	Shown(held []string) []*playerv1.Title
	Worn(held []string, choice string) *playerv1.Title
}

type Accounts interface {
	CreatedAt(ctx context.Context, account cpsession.AccountID) (time.Time, error)
}

var ErrNoPlayer = errors.New("no player has this name")

func NewPostgresQuery(db cppg.Querier, titles Titles, accounts Accounts, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, accounts: accounts, clock: clock}
}

type PostgresQuery struct {
	db       cppg.Querier
	titles   Titles
	accounts Accounts
	clock    cptime.Clock
}

const player = `
	SELECT
		profiles.account_id,
		profiles.name,
		profiles.admin,
		profiles.color,
		COALESCE(stats.tiles_taken, 0),
		CASE WHEN stats.streak_last_day IS NULL OR stats.streak_last_day IN ($2::date, $2::date - 1)
			THEN COALESCE(stats.streak_current, 0) ELSE 0 END,
		COALESCE(stats.streak_best, 0),
		COALESCE(to_char(stats.streak_last_day, 'YYYY-MM-DD'), ''),
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

func (q *PostgresQuery) Player(ctx context.Context, name string) (*playerv1.GetPlayerResponse, error) {
	var (
		account uuid.UUID
		color   int32
		choice  string
		held    []string
		answer  = &playerv1.Player{Stats: &playerv1.Stats{}}
	)
	err := q.db.QueryRowContext(ctx, player, folded(name), q.clock.Now().UTC().Format(time.DateOnly)).Scan(
		&account, &answer.Name, &answer.Admin, &color,
		&answer.Stats.TilesTaken, &answer.Stats.StreakCurrent, &answer.Stats.StreakBest, &answer.Stats.StreakLastDay,
		&choice, pq.Array(&held),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoPlayer
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the player: %w", err)
	}

	if answer.Color, err = playerread.KeptColor(color); err != nil {
		return nil, fmt.Errorf("failed to read the player's color: %w", err)
	}

	createdAt, err := q.accounts.CreatedAt(ctx, cpsession.AccountID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to ask when the account was made: %w", err)
	}
	if !createdAt.IsZero() {
		answer.CreatedAtUnixMs = createdAt.UnixMilli()
	}

	answer.Titles = q.titles.Shown(held)
	answer.WornTitle = q.titles.Worn(held, choice)
	return &playerv1.GetPlayerResponse{Player: answer}, nil
}

// The same fold the profile was kept under, or a name typed in another case finds nobody.
func folded(name string) string {
	return norm.NFKC.String(cases.Fold().String(norm.NFKD.String(strings.Trim(norm.NFC.String(name), " "))))
}
