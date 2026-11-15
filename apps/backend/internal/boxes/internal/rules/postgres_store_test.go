package rules_test

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)

	suite.Run(t, &cprules.StoreContractSuite[*boxesv1.Rules]{
		NewStore: func() cprules.Store[*boxesv1.Rules] {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresStore(db, parts.All())
		},
		Parts:   parts.All(),
		Default: rules.Seeded(),
		Rush: func() *boxesv1.Rules {
			return &boxesv1.Rules{MaxBoxesPerHour: proto.Uint32(30)}
		},
		Other: func() *boxesv1.Rules {
			return &boxesv1.Rules{MissRetry: durationpb.New(time.Minute)}
		},
	})
}

// With no bonuses schema to copy from, the migration seeds production's default.
func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAFrozenSetIsReadFrozen(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO rule_sets (name, rules) VALUES ('frozen', '{"frozen": true}')`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), "frozen")
	require.NoError(t, err)

	inForce := cprules.NewInForce[*boxesv1.Rules](cptime.SystemClock{})
	inForce.Enforce(set)
	assert.True(t, cprules.Reader(inForce, boxes.Part)().Frozen)
}

// The bonuses module's sets as production kept them on 15 November 2026, each with its box half and its reward half.
var bonusesSets = fstest.MapFS{
	"20261111200000_from_planet.up.sql": {Data: []byte(`
CREATE TABLE rule_sets (name text PRIMARY KEY, rules jsonb NOT NULL);
INSERT INTO rule_sets (name, rules) VALUES
	('default', '{"boxes": {"minInterval": "240s", "maxInterval": "480s", "maxChargesPerHour": 12},
		"kinds": {"refill": 5, "spreadClicks": 3, "encloseClicks": 2, "bomb": 1, "shields": 4},
		"offerTtl": "15s", "missRetry": "120s", "activeWithin": "120s", "shield": {"minPerBox": 10, "maxPerBox": 30},
		"welcome": {"within": "600s", "boxAfter": "60s"}, "signInRefill": true, "frozen": false}'),
	('bridge', '{"kinds": {"shields": 3}, "shield": {"minPerBox": 5, "maxPerBox": 20}}'),
	('capital-siege', '{"boxes": {"minInterval": "20s", "maxInterval": "40s", "maxChargesPerHour": 9999},
		"kinds": {"refill": 1, "spreadClicks": 0}, "missRetry": "30s", "welcome": {"boxAfter": "30s"}}'),
	('frozen', '{"kinds": {"shields": 3}, "frozen": true}'),
	('urf', '{"boxes": {"minInterval": "30s", "maxInterval": null}, "missRetry": "30s"}');
`)},
	"20261111200000_from_planet.down.sql": {Data: []byte(`DROP TABLE rule_sets;`)},
}

func TestTheMigrationCopiesEachOfTheBonusesSetsBoxHalfUnderItsName(t *testing.T) {
	database := server.CreateDatabase(t)
	database.OpenSchema(t, "bonuses", bonusesSets)
	db := database.OpenSchema(t, "boxes", migrations.FS)

	kept := map[string]string{}
	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT name, rules::text FROM rule_sets`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name, rules string
		require.NoError(t, rows.Scan(&name, &rules))
		kept[name] = rules
	}
	require.NoError(t, rows.Err())

	assert.JSONEq(t, `{"minInterval": "240s", "maxInterval": "480s", "maxBoxesPerHour": 12, "offerTtl": "15s",
		"missRetry": "120s", "activeWithin": "120s", "welcome": {"within": "600s", "boxAfter": "60s"}, "frozen": false}`,
		kept["default"])
	assert.JSONEq(t, `{}`, kept["bridge"], "a set that changes nothing of the boxes plays default's")
	assert.JSONEq(t, `{"minInterval": "20s", "maxInterval": "40s", "maxBoxesPerHour": 9999, "missRetry": "30s",
		"welcome": {"boxAfter": "30s"}}`, kept["capital-siege"])
	assert.JSONEq(t, `{"frozen": true}`, kept["frozen"])
	assert.JSONEq(t, `{"minInterval": "30s", "missRetry": "30s"}`, kept["urf"], "a null is left out")

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), "capital-siege")
	require.NoError(t, err)
	assert.Equal(t, uint32(9999), set.TakePlayedSnapshot().Rules.GetMaxBoxesPerHour())
}
