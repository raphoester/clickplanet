package migrations_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/migrationcheck/internal/migrations"
)

const auth = "internal/auth/internal/migrations"

func pair(dir, migration string) []string {
	return []string{dir + "/" + migration + ".up.sql", dir + "/" + migration + ".down.sql"}
}

func tree(dir string, migrationNames ...string) migrations.Tree {
	paths := make([]string, 0, 2*len(migrationNames))
	for _, m := range migrationNames {
		paths = append(paths, pair(dir, m)...)
	}
	return migrations.NewTree(paths)
}

func TestAnOrderedTreeHasNoProblem(t *testing.T) {
	ran := tree(auth, "20261111120000_recruitments")
	head := tree(auth, "20261111120000_recruitments", "20261111140000_arrival_tag")

	assert.Empty(t, migrations.Problems(ran, head))
}

func TestTheTreeKeepsOnlyTheMigrationsEachModuleEmbeds(t *testing.T) {
	head, err := migrations.TreeIn(fstest.MapFS{
		auth + "/20261111120000_recruitments.up.sql":   {},
		auth + "/20261111120000_recruitments.down.sql": {},
		auth + "/migrations.go":                        {},
		"internal/auth/internal/arrivals/notes.sql":    {},
		"migrations/20261111120000_stray.up.sql":       {},
	})

	require.NoError(t, err)
	assert.Empty(t, migrations.Problems(migrations.Tree{}, head))
}

func TestAFileGolangMigrateCannotReadIsAProblem(t *testing.T) {
	for _, file := range []string{
		"2026111112000_short.up.sql",
		"20261111120000_Upper.up.sql",
		"20261111120000_no_direction.sql",
		"recruitments.up.sql",
	} {
		problems := migrations.Problems(migrations.Tree{}, migrations.NewTree([]string{auth + "/" + file}))

		require.Lenf(t, problems, 1, "%s", file)
		assert.Equal(t, auth, problems[0].Directory)
		assert.Contains(t, problems[0].Reason, file)
	}
}

func TestAVersionThatIsNoDateIsAProblem(t *testing.T) {
	problems := migrations.Problems(migrations.Tree{}, tree(auth, "20261399120000_recruitments"))

	require.Len(t, problems, 2)
	assert.Contains(t, problems[0].Reason, "not a UTC date")
}

func TestTwoMigrationsOnOneVersionAreAProblem(t *testing.T) {
	head := tree(auth, "20261111120000_recruitments", "20261111120000_arrival_tag")

	problems := migrations.Problems(migrations.Tree{}, head)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Reason, "version 20261111120000 is 2 migrations")
	assert.Contains(t, problems[0].Reason, "refuses to start")
}

func TestAVersionWithNoUpIsAProblem(t *testing.T) {
	head := migrations.NewTree([]string{auth + "/20261111120000_recruitments.down.sql"})

	problems := migrations.Problems(migrations.Tree{}, head)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Reason, "no .up.sql")
}

func TestAMigrationOlderThanTheNewestThatRanIsAProblem(t *testing.T) {
	ran := tree(auth, "20261111140000_arrival_tag")
	head := tree(auth, "20261111120000_recruitments", "20261111140000_arrival_tag")

	problems := migrations.Problems(ran, head)

	require.Len(t, problems, 1)
	assert.Equal(t, auth, problems[0].Directory)
	assert.Contains(t, problems[0].Reason, "20261111120000_recruitments is new, but older than 20261111140000_arrival_tag")
	assert.Contains(t, problems[0].Reason, "would skip it")
}

func TestTheNewestThatRanGoneIsAProblem(t *testing.T) {
	ran := tree(auth, "20261111120000_recruitments", "20261111140000_arrival_tag")
	head := tree(auth, "20261111120000_recruitments")

	problems := migrations.Problems(ran, head)

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Reason, "20261111140000_arrival_tag is gone")
}

func TestAnOlderMigrationGoneOrRenamedIsNoProblem(t *testing.T) {
	ran := tree(auth, "20261111120000_recruitments", "20261111140000_arrival_tag")

	assert.Empty(t, migrations.Problems(ran, tree(auth, "20261111140000_arrival_tag")))
	assert.Empty(t, migrations.Problems(ran, tree(auth, "20261111120000_recruits", "20261111140000_arrival_tag")))
}

func TestAModuleAddedOrRemovedIsNoProblem(t *testing.T) {
	ran := tree("internal/seasons/internal/migrations", "20261111140000_rounds")
	head := tree("internal/scoring/internal/migrations", "20261111120000_rounds")

	assert.Empty(t, migrations.Problems(ran, head))
}
