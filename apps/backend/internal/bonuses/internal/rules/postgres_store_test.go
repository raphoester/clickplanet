package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)

	suite.Run(t, &cprules.StoreContractSuite[*bonusesv1.Rules]{
		NewStore: func() cprules.Store[*bonusesv1.Rules] {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresStore(db, parts.All())
		},
		Parts:   parts.All(),
		Default: rules.Seeded(),
		Rush: func() *bonusesv1.Rules {
			return &bonusesv1.Rules{Spread: &bonusesv1.SpreadRules{MaxPerBox: proto.Uint32(3)}}
		},
		Other: func() *bonusesv1.Rules {
			return &bonusesv1.Rules{Enclose: &bonusesv1.EncloseRules{MaxPerBox: proto.Uint32(5)}}
		},
	})
}

// With no planet schema to copy from, the migration seeds production's default.
func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAValueThisBuildDoesNotKnowIsIgnored(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{paused}', 'true') WHERE name = 'default'`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules))
}

func TestASetKeptThatCannotPlayIsRefused(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{spread,maxPerBox}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "spread.maxPerBox is 0")
}

func TestASetIsWrittenWithTheValuesItHoldsZerosIncluded(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)
	quiet, err := cprules.Snapshot[*bonusesv1.Rules]{Name: "quiet", Rules: &bonusesv1.Rules{
		Quiz: &bonusesv1.QuizRules{Enabled: proto.Bool(false)},
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, cprules.NewPostgresStore(db, parts.All()).Save(t.Context(), quiet))

	var kept string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules FROM rule_sets WHERE name = 'quiet'`).Scan(&kept))
	assert.JSONEq(t, `{"quiz": {"enabled": false}}`, kept)
}

func TestAFrozenSetIsReadFrozen(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO rule_sets (name, rules) VALUES ('frozen', '{"frozen": true}')`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), "frozen")
	require.NoError(t, err)

	inForce := cprules.NewInForce[*bonusesv1.Rules](cptime.SystemClock{})
	inForce.Enforce(set)
	assert.True(t, cprules.Reader(inForce, bonuses.Part)().Frozen)
}
