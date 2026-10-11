package account_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoAccount = errors.New("no account is shown under this name")

// What DisplayNameOf puts before a guest's code. No username folds to a name that starts with it.
const guestPrefix = "guest_"

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const account = `
	SELECT account_id FROM profiles WHERE name_folded = $1
	UNION ALL
	SELECT account_id FROM guest_codes WHERE code = $2
	LIMIT 1
`

func (q *PostgresQuery) Account(ctx context.Context, name string) (*playerv1.FindAccountResponse, error) {
	answer := &playerv1.FindAccountResponse{}
	err := q.db.QueryRowContext(ctx, account, playerread.FoldedName(name), guestCode(name)).Scan(&answer.AccountId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoAccount
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the account shown under a name: %w", err)
	}
	return answer, nil
}

func guestCode(name string) sql.NullString {
	code, guest := strings.CutPrefix(strings.Trim(name, " "), guestPrefix)
	return sql.NullString{String: code, Valid: guest}
}
