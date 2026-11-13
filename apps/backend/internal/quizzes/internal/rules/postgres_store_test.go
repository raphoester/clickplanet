package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "quizzes", migrations.FS)

	suite.Run(t, &cprules.StoreContractSuite[*quizzesv1.Rules]{
		NewStore: func() cprules.Store[*quizzesv1.Rules] {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresStore(db, parts.All())
		},
		Parts:   parts.All(),
		Default: rules.Seeded(),
		Rush: func() *quizzesv1.Rules {
			return &quizzesv1.Rules{MaxRightAnswersPerHour: proto.Uint32(30)}
		},
		Other: func() *quizzesv1.Rules {
			return &quizzesv1.Rules{LeaderBias: proto.Float64(2)}
		},
	})
}

// With no bonuses schema to copy from, the migration seeds production's default.
func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "quizzes", migrations.FS)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAFrozenSetIsReadFrozen(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "quizzes", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO rule_sets (name, rules) VALUES ('frozen', '{"frozen": true}')`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), "frozen")
	require.NoError(t, err)

	inForce := cprules.NewInForce[*quizzesv1.Rules](cptime.SystemClock{})
	inForce.Enforce(set)
	assert.True(t, cprules.Reader(inForce, quizzes.Part)().Frozen)
}
