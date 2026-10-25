package rule_sets_query

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Sets(ctx context.Context) ([]*planetv1.RuleSet, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT name, tempo, box_min_interval_ms, box_max_interval_ms, box_max_charges_per_hour, crowd
		FROM rule_sets ORDER BY name <> 'default', name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sets []*planetv1.RuleSet
	for rows.Next() {
		var (
			name                     string
			tempo                    float64
			minInterval, maxInterval int64
			maxCharges, crowd        uint32
		)
		if err := rows.Scan(&name, &tempo, &minInterval, &maxInterval, &maxCharges, &crowd); err != nil {
			return nil, fmt.Errorf("failed to read a rule set: %w", err)
		}
		sets = append(sets, &planetv1.RuleSet{
			Name:  name,
			Tempo: tempo,
			Boxes: &planetv1.BoxPace{
				MinInterval:       durationpb.New(time.Duration(minInterval) * time.Millisecond),
				MaxInterval:       durationpb.New(time.Duration(maxInterval) * time.Millisecond),
				MaxChargesPerHour: maxCharges,
			},
			AntiBot: &planetv1.AntiBotRules{Crowd: crowd},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}

	return sets, nil
}
