package cppg_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePasswordStaysOutOfAPrintedConfig(t *testing.T) {
	config := struct{ Database cppg.Config }{cppg.Config{Host: "postgres", Password: "hunter2"}}

	printed := fmt.Sprintf("%+v", config)

	assert.NotContains(t, printed, "hunter2")
	assert.Contains(t, printed, "postgres")
}

func TestAnIncompleteConfigNamesEveryMissingKey(t *testing.T) {
	err := cppg.Config{Host: "postgres", Port: "5432"}.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "[user dbName sslMode schema] is empty")
}

func TestASchemaThatIsNotAPlainIdentifierIsRefused(t *testing.T) {
	config := cppg.Config{Host: "h", Port: "5432", User: "u", DBName: "d", SSLMode: "disable", Schema: "planet; drop"}

	require.ErrorContains(t, config.Validate(), "not a lowercase identifier")
}

func TestACompleteConfigIsValid(t *testing.T) {
	require.NoError(t, cppg.Config{Host: "h", Port: "5432", User: "u", DBName: "d", SSLMode: "disable", Schema: "planet"}.Validate())
}

func TestEachSchemaHoldsItsOwnTablesAndMigrationHistory(t *testing.T) {
	migrations := fstest.MapFS{
		"1_create_things.up.sql":   {Data: []byte(`CREATE TABLE things (id integer PRIMARY KEY)`)},
		"1_create_things.down.sql": {Data: []byte(`DROP TABLE things`)},
	}
	ctx := t.Context()

	server := cppg.StartTestServer(t)
	first := server.OpenSchema(t, "first", migrations)
	second := server.OpenSchema(t, "second", migrations)

	_, err := first.ExecContext(ctx, `INSERT INTO things VALUES (1)`)
	require.NoError(t, err)

	var count int
	require.NoError(t, second.QueryRowContext(ctx, `SELECT count(*) FROM things`).Scan(&count))
	assert.Zero(t, count, "the same unqualified table name is a different table in each schema")

	var inPublic *string
	require.NoError(t, first.QueryRowContext(ctx, `SELECT to_regclass('public.things')::text`).Scan(&inPublic))
	assert.Nil(t, inPublic, "nothing lands in public")

	var version int
	require.NoError(t, second.QueryRowContext(ctx, `SELECT version FROM second.schema_migrations`).Scan(&version))
	assert.Equal(t, 1, version)
}
