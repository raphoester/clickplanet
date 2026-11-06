package renames_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

// Which renames: one account's, or everybody's for NoAccount, made at Since or after.
type Filter struct {
	Account cpsession.AccountID
	Since   time.Time
	Limit   uint32
}

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// $1 is the account, NULL for every account's, $2 the start, $3 the limit.
const renames = `
	SELECT id, account_id, renamed_at, old_name, new_name, reason, operator_id, client
	FROM renames
	WHERE ($1::uuid IS NULL OR account_id = $1) AND renamed_at >= $2
	ORDER BY renamed_at DESC, id DESC
	LIMIT $3
`

func (q *PostgresQuery) Renames(ctx context.Context, filter Filter) (*playerv1.ListRenamesResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, renames, accountOf(filter.Account), filter.Since.UTC(),
		limitOf(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read the renames: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &playerv1.ListRenamesResponse{Renames: []*playerv1.Rename{}}
	for rows.Next() {
		var (
			id       int64
			account  uuid.UUID
			at       time.Time
			rename   = &playerv1.Rename{}
			operator uuid.NullUUID
		)
		if err := rows.Scan(&id, &account, &at, &rename.OldName, &rename.NewName, &rename.Reason, &operator,
			&rename.Client); err != nil {
			return nil, fmt.Errorf("failed to scan a rename: %w", err)
		}
		rename.RenameId = fmt.Sprint(id)
		rename.AccountId = account.String()
		rename.RenamedAt = timestamppb.New(at)
		if operator.Valid {
			rename.OperatorAccountId = operator.UUID.String()
		}
		answer.Renames = append(answer.Renames, rename)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the renames: %w", err)
	}
	return answer, nil
}

func accountOf(account cpsession.AccountID) uuid.NullUUID {
	if account == cpsession.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: true}
}

func limitOf(asked uint32) uint32 {
	switch {
	case asked == 0:
		return DefaultLimit
	case asked > MaxLimit:
		return MaxLimit
	default:
		return asked
	}
}
