package postgres_rules_store

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var (
	_ rules.Store          = (*Store)(nil)
	_ flagmissions.Sizings = (*Store)(nil)
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

// The one document of the rules table, which migration 20261106180000_rules seeds.
type Store struct {
	db cppg.Conn
}

func (s *Store) Rules(ctx context.Context) (rules.Rules, error) {
	var kept []byte
	if err := s.db.On(ctx).QueryRowContext(ctx, `SELECT rules FROM rules`).Scan(&kept); err != nil {
		return rules.Rules{}, fmt.Errorf("failed to read the missions' rules: %w", err)
	}
	var message missionsv1.Rules
	if err := protojson.Unmarshal(kept, &message); err != nil {
		return rules.Rules{}, fmt.Errorf("the missions' rules kept are no missions.v1.Rules: %w", err)
	}
	return rules.RulesOf(&message) //nolint:wrapcheck // it names every number that cannot play.
}

func (s *Store) Sizing(ctx context.Context) (flagmissions.Sizing, error) {
	kept, err := s.Rules(ctx)
	if err != nil {
		return flagmissions.Sizing{}, err
	}
	return kept.FlagMissions(), nil
}

func (s *Store) Save(ctx context.Context, kept rules.Rules) error {
	document, err := protojson.Marshal(kept.Snapshot())
	if err != nil {
		return fmt.Errorf("failed to write the missions' rules: %w", err)
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rules (rules) VALUES ($1) ON CONFLICT (singleton) DO UPDATE SET rules = EXCLUDED.rules
	`, string(document)); err != nil {
		return fmt.Errorf("failed to keep the missions' rules: %w", err)
	}
	return nil
}
