package accesslog_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-ops/internal/accesslog"
	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

var noon = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

var roomy = excerpt.Limit{Lines: 100, Bytes: 1 << 20}

func entry(at time.Time, path string) string {
	return fmt.Sprintf(`{"level":"info","ts":%d.5,"request":{"uri":%q}}`, at.Unix(), path)
}

func writeLog(t *testing.T, directory, name string, lastWrite time.Time, lines ...string) {
	t.Helper()

	content := []byte(strings.Join(lines, "\n") + "\n")
	if strings.HasSuffix(name, ".gz") {
		var zipped bytes.Buffer
		writer := gzip.NewWriter(&zipped)
		_, err := writer.Write(content)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		content = zipped.Bytes()
	}

	path := filepath.Join(directory, name)
	require.NoError(t, os.WriteFile(path, content, 0o600))
	require.NoError(t, os.Chtimes(path, lastWrite, lastWrite))
}

func TestARotatedFileAndTheLiveOneAreReadOldestFirst(t *testing.T) {
	directory := t.TempDir()
	writeLog(
		t, directory, "access.log", noon.Add(2*time.Hour),
		entry(noon.Add(90*time.Minute), "/third"),
	)
	writeLog(
		t, directory, "access-2026-10-08T13-00-00.000.log.gz", noon.Add(time.Hour),
		entry(noon.Add(10*time.Minute), "/first"),
		entry(noon.Add(20*time.Minute), "/second"),
	)

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(3 * time.Hour), Limit: roomy,
	})

	require.NoError(t, err)
	assert.False(t, found.Truncated)
	require.Len(t, found.Lines, 3)
	assert.Contains(t, found.Lines[0], "/first")
	assert.Contains(t, found.Lines[1], "/second")
	assert.Contains(t, found.Lines[2], "/third")
}

func TestARequestOutsideTheWindowIsLeftOut(t *testing.T) {
	directory := t.TempDir()
	writeLog(
		t, directory, "access.log", noon.Add(2*time.Hour),
		entry(noon.Add(-time.Minute), "/before"),
		entry(noon.Add(time.Minute), "/inside"),
		entry(noon.Add(61*time.Minute), "/after"),
	)

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(time.Hour), Limit: roomy,
	})

	require.NoError(t, err)
	require.Len(t, found.Lines, 1)
	assert.Contains(t, found.Lines[0], "/inside")
}

func TestOnlyTheRequestsThatHoldTheTextAreKept(t *testing.T) {
	directory := t.TempDir()
	writeLog(
		t, directory, "access.log", noon.Add(time.Hour),
		entry(noon.Add(time.Minute), "/planet.v1.ClickService/Click"),
		entry(noon.Add(2*time.Minute), "/chat.v1.ChatService/SendMessage"),
	)

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(time.Hour), Contains: "ChatService", Limit: roomy,
	})

	require.NoError(t, err)
	require.Len(t, found.Lines, 1)
	assert.Contains(t, found.Lines[0], "SendMessage")
}

func TestAFileLastWrittenBeforeTheWindowIsNotOpened(t *testing.T) {
	directory := t.TempDir()
	writeLog(t, directory, "access.log", noon.Add(time.Hour), entry(noon.Add(time.Minute), "/inside"))
	stale := filepath.Join(directory, "access-2026-10-01T00-00-00.000.log.gz")
	require.NoError(t, os.WriteFile(stale, []byte("not a gzip stream"), 0o600))
	require.NoError(t, os.Chtimes(stale, noon.Add(-24*time.Hour), noon.Add(-24*time.Hour)))

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(time.Hour), Limit: roomy,
	})

	require.NoError(t, err)
	assert.Len(t, found.Lines, 1)
}

func TestAFileBegunAfterTheWindowIsNotOpened(t *testing.T) {
	directory := t.TempDir()
	writeLog(
		t, directory, "access-2026-10-08T13-00-00.000.log.gz", noon.Add(time.Hour),
		entry(noon.Add(time.Minute), "/inside"),
	)
	later := filepath.Join(directory, "access-2026-10-09T13-00-00.000.log.gz")
	require.NoError(t, os.WriteFile(later, []byte("not a gzip stream"), 0o600))
	require.NoError(t, os.Chtimes(later, noon.Add(25*time.Hour), noon.Add(25*time.Hour)))

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(30 * time.Minute), Limit: roomy,
	})

	require.NoError(t, err)
	assert.Len(t, found.Lines, 1)
}

func TestMoreRequestsThanTheLimitAreCutAndSaidSo(t *testing.T) {
	directory := t.TempDir()
	writeLog(
		t, directory, "access.log", noon.Add(time.Hour),
		entry(noon.Add(time.Minute), "/first"),
		entry(noon.Add(2*time.Minute), "/second"),
		entry(noon.Add(3*time.Minute), "/third"),
	)

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(time.Hour), Limit: excerpt.Limit{Lines: 2, Bytes: 1 << 20},
	})

	require.NoError(t, err)
	assert.True(t, found.Truncated)
	assert.Len(t, found.Lines, 2)
}

func TestALineThatIsNotALogEntryIsLeftOut(t *testing.T) {
	directory := t.TempDir()
	writeLog(
		t, directory, "access.log", noon.Add(time.Hour),
		"a line cut short by a crash",
		entry(noon.Add(time.Minute), "/inside"),
	)

	found, err := accesslog.NewDirectory(directory).Read(t.Context(), accesslog.Query{
		Since: noon, Until: noon.Add(time.Hour), Limit: roomy,
	})

	require.NoError(t, err)
	assert.Len(t, found.Lines, 1)
}
