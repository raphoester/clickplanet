package journal_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
	"github.com/raphoester/clickplanet.lol-ops/internal/journal"
)

var roomy = excerpt.Limit{Lines: 100, Bytes: 1 << 20}

func installJournalctl(t *testing.T, script string) (argumentsFile string) {
	t.Helper()

	directory := t.TempDir()
	argumentsFile = filepath.Join(directory, "arguments")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argumentsFile + "\n" + script + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(directory, "journalctl"), []byte(body), 0o700))
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argumentsFile
}

func TestATagIsOneOfThisStacksContainers(t *testing.T) {
	for _, raw := range []string{"cp-backend", "cp-postgres", "cp-metrics-poller"} {
		tag, err := journal.ParseTag(raw)

		require.NoError(t, err)
		assert.Equal(t, journal.Tag(raw), tag)
	}
}

func TestATagOfAnythingElseIsRefused(t *testing.T) {
	for _, raw := range []string{"", "sshd", "cp-", "cp-backend --since", "cp-backend\n_UID=0", "CP-BACKEND"} {
		_, err := journal.ParseTag(raw)

		require.ErrorIs(t, err, journal.ErrForeignTag, raw)
	}
}

func TestJournalctlIsAskedForOneContainerOverTheWindow(t *testing.T) {
	argumentsFile := installJournalctl(t, `echo 'level=INFO msg="loaded the tile map"'`)
	since := time.Date(2026, 10, 8, 8, 0, 0, 0, time.UTC)

	found, err := journal.NewJournalctl("/var/log/journal").Read(t.Context(), journal.Query{
		Tag: "cp-backend", Since: since, Until: since.Add(time.Hour), Limit: roomy,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{`level=INFO msg="loaded the tile map"`}, found.Lines)
	asked, err := os.ReadFile(argumentsFile)
	require.NoError(t, err)
	assert.Equal(t, `--directory
/var/log/journal
--quiet
--no-pager
--output
cat
--since
@1791446400
--until
@1791450000
CONTAINER_TAG=cp-backend
`, string(asked))
}

func TestOnlyTheLinesThatHoldTheTextAreKept(t *testing.T) {
	installJournalctl(t, `printf '%s\n' 'antibot ban scope=203.0.113.7' 'loaded the tile map' 'antibot ban scope=198.51.100.4'`)

	found, err := journal.NewJournalctl("/var/log/journal").Read(t.Context(), journal.Query{
		Tag: "cp-backend", Contains: "antibot ban", Limit: roomy,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"antibot ban scope=203.0.113.7", "antibot ban scope=198.51.100.4"}, found.Lines)
}

func TestAJournalLongerThanTheLimitIsCutWithoutWaitingForItsEnd(t *testing.T) {
	installJournalctl(t, `while true; do echo 'one more line'; done`)

	found, err := journal.NewJournalctl("/var/log/journal").Read(t.Context(), journal.Query{
		Tag: "cp-backend", Limit: excerpt.Limit{Lines: 3, Bytes: 1 << 20},
	})

	require.NoError(t, err)
	assert.True(t, found.Truncated)
	assert.Len(t, found.Lines, 3)
}

func TestWhatJournalctlComplainsAboutIsReported(t *testing.T) {
	installJournalctl(t, `echo 'Failed to open /var/log/journal: Permission denied' >&2; exit 1`)

	_, err := journal.NewJournalctl("/var/log/journal").Read(t.Context(), journal.Query{
		Tag: "cp-backend", Limit: roomy,
	})

	require.ErrorContains(t, err, "Permission denied")
}

func TestAnEmptyWindowIsAnEmptyExcerpt(t *testing.T) {
	installJournalctl(t, `exit 0`)

	found, err := journal.NewJournalctl("/var/log/journal").Read(t.Context(), journal.Query{
		Tag: "cp-backend", Limit: roomy,
	})

	require.NoError(t, err)
	assert.Empty(t, found.Lines)
	assert.False(t, found.Truncated)
}
