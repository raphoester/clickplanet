package allegiance_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Flags is planet's catalog: a closed flag is not playable once no lock holds it.
type Flags interface {
	Playable(ctx context.Context, flag string) (bool, error)
}

func NewPostgresQuery(db cppg.Conn, lock func() allegiances.Lock, flags Flags) *PostgresQuery {
	return &PostgresQuery{db: db, lock: lock, flags: flags}
}

type PostgresQuery struct {
	db    cppg.Conn
	lock  func() allegiances.Lock
	flags Flags
}

func (q *PostgresQuery) Allegiance(ctx context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error) {
	if account == allegiances.NoAccount {
		return &planetv1.GetAllegianceResponse{}, nil
	}

	var (
		flag      string
		changedAt sql.NullTime
	)
	err := q.db.On(ctx).QueryRowContext(ctx, `SELECT flag, changed_at FROM allegiances WHERE account = $1`, string(account)).
		Scan(&flag, &changedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return &planetv1.GetAllegianceResponse{}, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read the allegiance of %s: %w", account, err)
	}

	playable, err := q.flags.Playable(ctx, flag)
	if err != nil {
		return nil, fmt.Errorf("failed to read whether %q may be played: %w", flag, err)
	}
	allegiance := allegiancemessage.Encode(flag, allegiances.ChangedAt(changedAt.Time).ChangeableAt(), bool(q.lock()))
	allegiance.Closed = !playable
	return &planetv1.GetAllegianceResponse{Allegiance: allegiance}, nil
}
