package postgres_rule_set_store_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &rules.StoreContractSuite{
		NewStore: func() rules.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_rule_set_store.New(db, parts.All())
		},
		Parts: parts.All(),
	})
}

func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	set, err := postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAValueThisBuildDoesNotKnowIsIgnored(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{paused}', 'true') WHERE name = 'default'`)
	require.NoError(t, err)

	set, err := postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules))
}

func TestASetKeptThatCannotPlayIsRefused(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{clicks,burst}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), rules.Default)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.burst")
}

func TestASetIsWrittenWithTheValuesItHoldsZerosIncluded(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	quiet, err := rules.Snapshot{Name: "quiet", Rules: &planetv1.Rules{
		Bonuses: &planetv1.BonusRules{Quiz: &planetv1.QuizRules{Enabled: proto.Bool(false)}},
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db, parts.All()).Save(t.Context(), quiet))

	assert.JSONEq(t, `{"bonuses": {"quiz": {"enabled": false}}}`, document(t, db, "quiet"))
}

func TestAFrozenSetIsReadFrozen(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO rule_sets (name, rules) VALUES ('frozen', '{"frozen": true}')`)
	require.NoError(t, err)

	set, err := postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), "frozen")
	require.NoError(t, err)

	inForce := rules.NewInForce(cptime.SystemClock{})
	inForce.Enforce(set)
	assert.Equal(t, clicks.Freeze(true), rules.Reader(inForce, clicks.FreezePart)())
}

const inheritDefault = "20261031180000"

func TestTheMigrationKeepsInEachSetWhatItChangesAndItPlaysAsBefore(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", cppg.MigrationsBefore(t, migrations.FS, inheritDefault))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules)
		SELECT 'rush', jsonb_set(jsonb_set(rules, '{clicks,tempo}', '3'), '{toll,steps}', '[]')
			|| '{"frozen": false, "allegianceLocked": true}'
		FROM rule_sets WHERE name = 'default';
		INSERT INTO rule_sets (name, rules)
		SELECT 'calm', jsonb_set(rules #- '{clicks,newAccountClicks}', '{bonuses,welcome}', 'null')
		FROM rule_sets WHERE name = 'default'`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.JSONEq(t, `{"clicks": {"tempo": 3}, "toll": {"steps": []}, "allegianceLocked": true}`, document(t, db, "rush"))
	assert.JSONEq(t, `{"clicks": {"newAccountClicks": 0}, "bonuses": {"welcome": {"within": "0s", "boxAfter": "0s"}}}`,
		document(t, db, "calm"))

	store := postgres_rule_set_store.New(db, parts.All())
	rush, err := store.Set(t.Context(), "rush")
	require.NoError(t, err)
	before := rules.Named("rush", func(r *planetv1.Rules) {
		r.Clicks.Tempo = proto.Float64(3)
		r.Toll.Steps = nil
		r.AllegianceLocked = proto.Bool(true)
	})
	assert.True(t, proto.Equal(before.Rules, rush.TakePlayedSnapshot().Rules), "%v", rush.TakePlayedSnapshot().Rules)
}

func TestTheMigrationGoesBackToSetsThatHoldEveryValue(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO rule_sets (name, rules) VALUES ('rush', '{"clicks": {"tempo": 3}, "toll": {"steps": []}}')`)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, inheritDefault+"_rule_sets_inherit_default.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	var played planetv1.Rules
	require.NoError(t, protojson.Unmarshal([]byte(document(t, db, "rush")), &played))
	assert.True(t, proto.Equal(rules.Named("rush", func(r *planetv1.Rules) {
		r.Clicks.Tempo = proto.Float64(3)
		r.Toll.Steps = nil
	}).Rules, &played), "%v", &played)
}

func document(t *testing.T, db *cppg.Postgres, name string) string {
	t.Helper()

	var kept string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules FROM rule_sets WHERE name = $1`, name).Scan(&kept))
	return kept
}
