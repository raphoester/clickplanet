package migrations_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	ruleKnobs   = "20261111150000"
	arenaFreeze = "20261111150200"
)

func rulesOf(t *testing.T, db *cppg.Postgres, name string) string {
	t.Helper()
	var rules string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules::text FROM rule_sets WHERE name = $1`, name).Scan(&rules))
	return rules
}

func TestTheCapitalShieldsBecomeTheCapitalsSeed(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, ruleKnobs))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules) VALUES
			('siege', '{"arena": {"on": true, "capitalShields": 40}}'),
			('urf', '{"clicks": {"tempo": 3}}')
	`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.JSONEq(t, `{"arena": {"on": true}, "seed": {"capitals": {"shields": 40}}}`, rulesOf(t, db, "siege"))
	assert.JSONEq(t, `{"clicks": {"tempo": 3}}`, rulesOf(t, db, "urf"), "a set that held none holds none")
	assert.Contains(t, rulesOf(t, db, "default"), `"seed": {"capitals": {"shields": 100}}`)
	assert.Contains(t, rulesOf(t, db, "default"), `"reach": false`)
	assert.Contains(t, rulesOf(t, db, "default"), `"capitals": {"knockOut": false, "fixedShields": false}`)
	assert.NotContains(t, rulesOf(t, db, "default"), "capitalShields")
}

func TestTheCapitalsSeedGoesBackToTheArena(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, arenaFreeze))
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules) VALUES
			('siege', '{"seed": {"capitals": {"shields": 40}}, "reach": true, "capitals": {"knockOut": true}}'),
			('urf', '{"clicks": {"tempo": 3}}')
	`)
	require.NoError(t, err)

	down, err := fs.ReadFile(migrations.FS, ruleKnobs+"_rule_knobs.down.sql")
	require.NoError(t, err)
	_, err = db.On(t.Context()).ExecContext(t.Context(), string(down))
	require.NoError(t, err)

	assert.JSONEq(t, `{"arena": {"capitalShields": 40}}`, rulesOf(t, db, "siege"))
	assert.JSONEq(t, `{"clicks": {"tempo": 3}}`, rulesOf(t, db, "urf"))
	assert.Contains(t, rulesOf(t, db, "default"), `"capitalShields": 100`)
	assert.NotContains(t, rulesOf(t, db, "default"), `"reach"`)
}

func TestAnArenaKeptBeforeItsFreezeFreezesEightSeconds(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", before(t, arenaFreeze))
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO arenas (id, ends_at, opened_at) VALUES ('battle', now() + interval '1 hour', now())`)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	assert.Equal(t, []int{8000}, scan(t, db, `SELECT freeze_ms FROM arenas`))
}
