package migrations_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Production's planet rule sets on 11 November 2026, as planet.v1.AdminService/GetRules answered them.
var productionsPlanetSets = map[string]string{
	"default": `{"clicks":{"tempo":1,"perSecond":0.2,"burst":60,"scopeMultiplier":10,"v6ScopeMultiplier":4,
		"guestScopeMultiplier":1,"linkedMultiplier":2,"newAccountClicks":60,"fastStart":"1800s"},
		"toll":{"steps":[{"share":0.1,"slowdown":1.5},{"share":0.2,"slowdown":2.5},{"share":0.3,"slowdown":4}]},
		"bonuses":{"boxes":{"minInterval":"240s","maxInterval":"480s","maxChargesPerHour":12},
		"kinds":{"refill":5,"spreadClicks":3,"encloseClicks":2,"bomb":1,"shields":4},"offerTtl":"15s","missRetry":"120s",
		"activeWithin":"120s","spread":{"clicks":8,"maxPerBox":4},"bomb":{"rings":4,"off":false},
		"enclose":{"maxTiles":25,"held":3,"maxPerBox":3,"unlimited":false},
		"shield":{"held":60,"minPerBox":10,"maxPerBox":30,"perTile":10},"welcome":{"within":"600s","boxAfter":"60s"},
		"quiz":{"enabled":true,"minInterval":"240s","maxInterval":"480s","missRetry":"120s","offerTtl":"25s",
		"answerWindow":"8s","leaderBias":1,"maxChargesPerHour":12},"signInRefill":true},
		"frozen":false,"allegianceLocked":false,"arena":{"on":false,"freeze":"8s"},"seed":{"capitals":{"shields":100}},
		"reach":false,"capitals":{"knockOut":false,"fixedShields":false}}`,
	"bridge": `{"bonuses":{"kinds":{"shields":3},"shield":{"held":30,"minPerBox":5,"maxPerBox":20}}}`,
	"capital-siege": `{"clicks":{"tempo":3},"toll":{"steps":[{"share":0.1,"slowdown":1.5},{"share":0.2,"slowdown":2.5},
		{"share":0.3,"slowdown":4},{"share":0.4,"slowdown":6},{"share":0.5,"slowdown":10}]},
		"bonuses":{"boxes":{"minInterval":"30s","maxInterval":"60s","maxChargesPerHour":9999},
		"kinds":{"refill":1,"spreadClicks":2,"encloseClicks":1,"shields":1},"missRetry":"30s",
		"spread":{"clicks":50,"maxPerBox":25},"bomb":{"rings":8,"off":true},"enclose":{"maxTiles":35,"held":8,"maxPerBox":5},
		"shield":{"held":75,"minPerBox":20,"maxPerBox":44},"welcome":{"boxAfter":"30s"},
		"quiz":{"minInterval":"30s","maxInterval":"60s","missRetry":"60s","maxChargesPerHour":9999}},
		"allegianceLocked":true,"arena":{"on":true,"freeze":"8s"},"seed":{"capitals":{"shields":100}},"reach":true,
		"capitals":{"knockOut":true,"fixedShields":true}}`,
	"frozen": `{"bonuses":{"kinds":{"shields":3},"shield":{"held":30,"minPerBox":5,"maxPerBox":20}},"frozen":true,
		"allegianceLocked":true}`,
	"unlimited-enclose": `{"clicks":{"tempo":3},"bonuses":{"kinds":{"encloseClicks":0},
		"enclose":{"maxTiles":1000000,"unlimited":true}},"allegianceLocked":true}`,
	"urf": `{"clicks":{"tempo":3},"toll":{"steps":[{"share":0.1,"slowdown":1.5},{"share":0.2,"slowdown":2.5},
		{"share":0.3,"slowdown":4},{"share":0.4,"slowdown":6},{"share":0.5,"slowdown":10}]},
		"bonuses":{"boxes":{"minInterval":"30s","maxInterval":"60s","maxChargesPerHour":9999},
		"kinds":{"refill":1,"spreadClicks":1,"encloseClicks":1,"shields":1},"missRetry":"30s",
		"spread":{"clicks":12,"maxPerBox":8},"bomb":{"rings":8},"enclose":{"maxTiles":35,"held":8,"maxPerBox":5},
		"shield":{"held":75,"minPerBox":20,"maxPerBox":44},"welcome":{"boxAfter":"30s"},
		"quiz":{"minInterval":"30s","maxInterval":"60s","missRetry":"60s","maxChargesPerHour":9999}},
		"allegianceLocked":true}`,
}

// What each set holds of the offers once it came across: every value of the offers' half, and frozen; nothing of the
// wallet's half (spread.clicks, bomb, enclose.maxTiles, held, unlimited, shield.held, perTile).
var theirOffers = map[string]string{
	"default": `{"boxes":{"minInterval":"240s","maxInterval":"480s","maxChargesPerHour":12},
		"kinds":{"refill":5,"spreadClicks":3,"encloseClicks":2,"bomb":1,"shields":4},"offerTtl":"15s","missRetry":"120s",
		"activeWithin":"120s","spread":{"maxPerBox":4},"enclose":{"maxPerBox":3},"shield":{"minPerBox":10,"maxPerBox":30},
		"welcome":{"within":"600s","boxAfter":"60s"},"quiz":{"enabled":true,"minInterval":"240s","maxInterval":"480s",
		"missRetry":"120s","offerTtl":"25s","answerWindow":"8s","leaderBias":1,"maxChargesPerHour":12},
		"signInRefill":true,"frozen":false}`,
	"bridge": `{"kinds":{"shields":3},"shield":{"minPerBox":5,"maxPerBox":20}}`,
	"capital-siege": `{"boxes":{"minInterval":"30s","maxInterval":"60s","maxChargesPerHour":9999},
		"kinds":{"refill":1,"spreadClicks":2,"encloseClicks":1,"shields":1},"missRetry":"30s","spread":{"maxPerBox":25},
		"enclose":{"maxPerBox":5},"shield":{"minPerBox":20,"maxPerBox":44},"welcome":{"boxAfter":"30s"},
		"quiz":{"minInterval":"30s","maxInterval":"60s","missRetry":"60s","maxChargesPerHour":9999}}`,
	"frozen":            `{"kinds":{"shields":3},"shield":{"minPerBox":5,"maxPerBox":20},"frozen":true}`,
	"unlimited-enclose": `{"kinds":{"encloseClicks":0}}`,
	"urf": `{"boxes":{"minInterval":"30s","maxInterval":"60s","maxChargesPerHour":9999},
		"kinds":{"refill":1,"spreadClicks":1,"encloseClicks":1,"shields":1},"missRetry":"30s","spread":{"maxPerBox":8},
		"enclose":{"maxPerBox":5},"shield":{"minPerBox":20,"maxPerBox":44},"welcome":{"boxAfter":"30s"},
		"quiz":{"minInterval":"30s","maxInterval":"60s","missRetry":"60s","maxChargesPerHour":9999}}`,
}

// Planet's table as its own migrations keep it; its migrations are planet's, which this module cannot import.
var planetRuleSets = fstest.MapFS{"20261026210000_rule_sets.up.sql": {Data: []byte(`
	CREATE TABLE rule_sets (name text PRIMARY KEY, rules jsonb NOT NULL)`)}}

func TestEachOfProductionsSetsComesAcrossWithItsOffersAndFrozen(t *testing.T) {
	server := cppg.StartTestServer(t)
	planet := server.OpenSchema(t, "planet", planetRuleSets)
	for name, kept := range productionsPlanetSets {
		_, err := planet.On(t.Context()).ExecContext(t.Context(),
			`INSERT INTO rule_sets (name, rules) VALUES ($1, $2)`, name, kept)
		require.NoError(t, err)
	}

	db := server.OpenSchema(t, "bonuses", migrations.FS)

	store := postgres_rule_set_store.New(db, parts.All())
	names, err := store.Names(t.Context())
	require.NoError(t, err)
	assert.Len(t, names, len(productionsPlanetSets))
	for name, offers := range theirOffers {
		var kept string
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT rules FROM rule_sets WHERE name = $1`, name).Scan(&kept))
		assert.JSONEq(t, offers, kept, name)

		_, err := store.Set(t.Context(), rules.SetName(name))
		assert.NoError(t, err, "%s plays over default", name)
	}
}

func TestAPlanetSetThatHoldsNoneOfTheOffersComesAcrossEmpty(t *testing.T) {
	server := cppg.StartTestServer(t)
	planet := server.OpenSchema(t, "planet", planetRuleSets)
	_, err := planet.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules) VALUES ('default', $1), ('calm', '{"clicks": {"tempo": 0.5}}')`,
		productionsPlanetSets["default"])
	require.NoError(t, err)

	db := server.OpenSchema(t, "bonuses", migrations.FS)

	var kept string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules FROM rule_sets WHERE name = 'calm'`).Scan(&kept))
	assert.JSONEq(t, `{}`, kept, "a set that changes none of the offers plays default's")
}
