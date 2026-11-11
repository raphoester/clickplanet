package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const chargesMove = "20261111200000"

// Production's sets on 11 November 2026, as planet.v1.AdminService/GetRules answered them, before the move.
var productionsSets = map[string]string{
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

// What each set holds of the wallet once the move ran: every value of its half, and nothing of the offers'.
var theirCharges = map[string]string{
	"default": `{"spread":{"clicks":8},"bomb":{"rings":4,"off":false},"enclose":{"maxTiles":25,"held":3,"unlimited":false},
		"shield":{"held":60,"perTile":10}}`,
	"bridge":            `{"shield":{"held":30}}`,
	"capital-siege":     `{"spread":{"clicks":50},"bomb":{"rings":8,"off":true},"enclose":{"maxTiles":35,"held":8},"shield":{"held":75}}`,
	"frozen":            `{"shield":{"held":30}}`,
	"unlimited-enclose": `{"enclose":{"maxTiles":1000000,"unlimited":true}}`,
	"urf":               `{"spread":{"clicks":12},"bomb":{"rings":8},"enclose":{"maxTiles":35,"held":8},"shield":{"held":75}}`,
}

func chargesOf(t *testing.T, db *cppg.Postgres, name string) string {
	t.Helper()
	var charges string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules -> 'charges' FROM rule_sets WHERE name = $1`, name).Scan(&charges))
	return charges
}

func TestEachOfProductionsSetsKeepsItsWalletUnderCharges(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, chargesMove))
	for name, kept := range productionsSets {
		_, err := db.On(t.Context()).ExecContext(t.Context(), `
			INSERT INTO rule_sets (name, rules) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules`,
			name, kept)
		require.NoError(t, err)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	store := postgres_rule_set_store.New(db, parts.All())
	for name, charges := range theirCharges {
		assert.JSONEq(t, charges, chargesOf(t, db, name), name)

		var bonuses string
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT rules -> 'bonuses' FROM rule_sets WHERE name = $1`, name).Scan(&bonuses))
		assert.NotEmpty(t, bonuses, "%s keeps bonuses for the offers' module to copy", name)

		_, err := store.Set(t.Context(), rules.SetName(name))
		assert.NoError(t, err, "%s plays over default", name)
	}
}

func TestASetThatHeldNoneOfTheWalletGetsNoCharges(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, chargesMove))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules) VALUES ('calm', '{"clicks": {"tempo": 0.5}, "bonuses": {"quiz": {"enabled": false}}}')`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.JSONEq(t, `{"clicks": {"tempo": 0.5}, "bonuses": {"quiz": {"enabled": false}}}`, rulesOf(t, db, "calm"))
}

func TestTheMoveGoesBackToBonusesAlone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, chargesMove))
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO rule_sets (name, rules) VALUES ('urf', $1)`, productionsSets["urf"])
	require.NoError(t, err)
	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	down, err := migrations.FS.ReadFile(chargesMove + "_charges.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.JSONEq(t, productionsSets["urf"], rulesOf(t, db, "urf"))
}
