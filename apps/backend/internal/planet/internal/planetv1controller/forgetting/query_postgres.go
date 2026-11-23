package forgetting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Forgetting names what the reverts forgot from the ledger's head on: a revert changes it, and so does the head passing
// one. It names no caller.
type Forgetting string

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// The flush writes a revert's mark and the forgotten column of its events in one transaction.
const marks = `
	SELECT 'scope:' || scope, before_position FROM ledger_forgotten
	UNION ALL
	SELECT 'account:' || account::text, before_position FROM ledger_forgotten_accounts
	ORDER BY 1
`

func (q *PostgresQuery) Forgetting(ctx context.Context) (Forgetting, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, marks)
	if err != nil {
		return "", fmt.Errorf("failed to read what the reverts forgot: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hash := sha256.New()
	for rows.Next() {
		var (
			caller string
			before int64
		)
		if err := rows.Scan(&caller, &before); err != nil {
			return "", fmt.Errorf("failed to scan what a revert forgot: %w", err)
		}
		_, _ = fmt.Fprintf(hash, "%s<%d;", caller, before)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("failed to read what the reverts forgot: %w", err)
	}
	return Forgetting(hex.EncodeToString(hash.Sum(nil))[:16]), nil
}
