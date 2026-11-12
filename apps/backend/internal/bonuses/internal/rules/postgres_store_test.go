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
