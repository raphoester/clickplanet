package rules_query

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// The document as postgres_rules_store keeps it.
func (q *PostgresQuery) Rules(ctx context.Context) (*missionsv1.GetRulesResponse, error) {
	var kept []byte
	if err := q.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rules`).Scan(&kept); err != nil {
		return nil, fmt.Errorf("failed to read the missions' rules: %w", err)
	}
	var message missionsv1.Rules
	if err := protojson.Unmarshal(kept, &message); err != nil {
		return nil, fmt.Errorf("the missions' rules kept are no missions.v1.Rules: %w", err)
	}
	return &missionsv1.GetRulesResponse{Rules: &message}, nil
}
