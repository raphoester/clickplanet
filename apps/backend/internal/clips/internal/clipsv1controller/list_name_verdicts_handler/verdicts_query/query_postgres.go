package verdicts_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/name_verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const limit = 1000

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const listed = `
	SELECT account, name, shown, reason, decided_by, decided_at
	FROM name_verdicts
	WHERE NOT ($1 AND shown)
	ORDER BY decided_at DESC, account
	LIMIT $2
`

func (q *PostgresQuery) Verdicts(ctx context.Context, hiddenOnly bool) (*clipsv1.ListNameVerdictsResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, listed, hiddenOnly, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read the verdicts of the names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := &clipsv1.ListNameVerdictsResponse{}
	for rows.Next() {
		var (
			account uuid.UUID
			by      string
			at      time.Time
			kept    verdicts.Snapshot
		)
		if err := rows.Scan(&account, &kept.Name, &kept.Shown, &kept.Reason, &by, &at); err != nil {
			return nil, fmt.Errorf("failed to read the verdict of a name: %w", err)
		}
		decider, err := verdicts.DeciderOf(by)
		if err != nil {
			return nil, err //nolint:wrapcheck // the domain named what is wrong.
		}
		kept.Account, kept.By, kept.At = verdicts.AccountID(account), decider, at.UTC()
		verdict, err := name_verdicts.ProtoOf(kept)
		if err != nil {
			return nil, err //nolint:wrapcheck // the domain named what is wrong.
		}
		out.Verdicts = append(out.Verdicts, verdict)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the verdicts of the names: %w", err)
	}
	return out, nil
}
