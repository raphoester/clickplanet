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
	"golang.org/x/sync/errgroup"
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

type Fronts interface {
	Fronts(ctx context.Context, account cpsession.AccountID) (*playerv1.GetFrontsResponse, error)
}

var ErrNoPlayer = errors.New("no player has this name")

const shownFronts = 3

func NewPostgresQuery(db cppg.Querier, titles Titles, accounts Accounts, fronts Fronts, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, accounts: accounts, fronts: fronts, clock: clock}
}

type PostgresQuery struct {
	db       cppg.Querier
	titles   Titles
	accounts Accounts
	fronts   Fronts
	clock    cptime.Clock
}

const player = `
	SELECT
		profiles.account_id,
		profiles.name,
		profiles.admin,
		profiles.color,
		COALESCE(stats.tiles_taken, 0),
		` + playerread.StreakNow + `,
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

	var createdAt time.Time
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		if createdAt, err = q.accounts.CreatedAt(groupCtx, cpsession.AccountID(account)); err != nil {
			return fmt.Errorf("failed to ask when the account was made: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		fronts, err := q.fronts.Fronts(groupCtx, cpsession.AccountID(account))
		if err != nil {
			return fmt.Errorf("failed to ask which countries the player plays for and against: %w", err)
		}
		answer.PlaysFor, answer.PlaysAgainst = topOf(fronts.GetPlaysFor()), topOf(fronts.GetPlaysAgainst())
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part already names what failed.
	}
	if !createdAt.IsZero() {
		answer.CreatedAtUnixMs = createdAt.UnixMilli()
	}

	answer.Titles = q.titles.Shown(held)
	answer.WornTitle = q.titles.Worn(held, choice)
	return &playerv1.GetPlayerResponse{Player: answer}, nil
}

func topOf(countries []*playerv1.CountryTiles) []*playerv1.CountryTiles {
	return countries[:min(len(countries), shownFronts)]
}

// The same fold the profile was kept under, or a name typed in another case finds nobody.
func folded(name string) string {
	return norm.NFKC.String(cases.Fold().String(norm.NFKD.String(strings.Trim(norm.NFC.String(name), " "))))
}
