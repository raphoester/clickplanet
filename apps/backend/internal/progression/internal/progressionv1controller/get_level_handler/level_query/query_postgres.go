package level_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Levels interface {
	Level(record progressionread.Record, kept uint32) *progressionv1.Level
	MissionRates() ([]*progressionv1.MissionRate, error)
	Unlocks() ([]*progressionv1.Unlock, error)
}

func NewPostgresQuery(db cppg.Conn, levels Levels) *PostgresQuery {
	return &PostgresQuery{db: db, levels: levels}
}

type PostgresQuery struct {
	db     cppg.Conn
	levels Levels
}

// A guest has no row: neither the raise nor the reconciliation keeps one.
const level = `
	SELECT ` + progressionread.RecordColumns + `, levels.level
	FROM levels
	LEFT JOIN stats ON stats.account_id = levels.account_id
	WHERE levels.account_id = $1
`

func (q *PostgresQuery) Level(ctx context.Context, account accounts.AccountID) (*progressionv1.GetLevelResponse, error) {
	rates, err := q.levels.MissionRates()
	if err != nil {
		return nil, fmt.Errorf("failed to tell the missions' rates: %w", err)
	}
	unlocks, err := q.levels.Unlocks()
	if err != nil {
		return nil, fmt.Errorf("failed to tell the ladder: %w", err)
	}

	var (
		record progressionread.Record
		kept   uint32
	)
	err = q.db.On(ctx).QueryRowContext(ctx, level, uuid.UUID(account)).Scan(append(record.Into(), &kept)...)
	if errors.Is(err, sql.ErrNoRows) {
		return &progressionv1.GetLevelResponse{MissionRates: rates, Unlocks: unlocks}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the level: %w", err)
	}
	return &progressionv1.GetLevelResponse{Level: q.levels.Level(record, kept), MissionRates: rates, Unlocks: unlocks}, nil
}
