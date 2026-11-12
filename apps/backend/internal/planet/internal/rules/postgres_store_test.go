package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &cprules.StoreContractSuite[*planetv1.Rules]{
		NewStore: func() cprules.Store[*planetv1.Rules] {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresStore(db, parts.All())
		},
		Parts:   parts.All(),
		Default: rules.Seeded(),
		Rush:    func() *planetv1.Rules { return &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}} },
		Other: func() *planetv1.Rules {
			return &planetv1.Rules{Charges: &planetv1.ChargeRules{Shield: &planetv1.ShieldRules{PerTile: proto.Uint32(20)}}}
		},
	})
}

func TestASetThatHoldsATollWithNoStepPlaysNone(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	free, err := cprules.Snapshot[*planetv1.Rules]{Name: "free", Rules: &planetv1.Rules{Toll: &planetv1.TollRules{}}}.
		Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	store := cprules.NewPostgresStore(db, parts.All())
	require.NoError(t, store.Save(t.Context(), free))

	set, err := store.Set(t.Context(), "free")
	require.NoError(t, err)

	assert.JSONEq(t, `{"toll": {}}`, document(t, db, "free"))
	assert.Empty(t, set.TakePlayedSnapshot().Rules.GetToll().GetSteps())
}

func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAValueThisBuildDoesNotKnowIsIgnored(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{paused}', 'true') WHERE name = 'default'`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules))
}

func TestASetKeptThatCannotPlayIsRefused(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{clicks,burst}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.burst")
}

func TestASetIsWrittenWithTheValuesItHoldsZerosIncluded(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	quiet, err := cprules.Snapshot[*planetv1.Rules]{Name: "quiet", Rules: &planetv1.Rules{
		Charges: &planetv1.ChargeRules{Bomb: &planetv1.BombRules{Off: proto.Bool(false)}},
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, cprules.NewPostgresStore(db, parts.All()).Save(t.Context(), quiet))

	assert.JSONEq(t, `{"charges": {"bomb": {"off": false}}}`, document(t, db, "quiet"))
}

func TestAFrozenSetIsReadFrozen(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO rule_sets (name, rules) VALUES ('frozen', '{"frozen": true}')`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), "frozen")
	require.NoError(t, err)

	inForce := cprules.NewInForce[*planetv1.Rules](cptime.SystemClock{})
	inForce.Enforce(set)
	assert.Equal(t, clicks.Freeze(true), cprules.Reader(inForce, clicks.FreezePart)())
}

func document(t *testing.T, db *cppg.Postgres, name string) string {
	t.Helper()

	var kept string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules FROM rule_sets WHERE name = $1`, name).Scan(&kept))
	return kept
}
