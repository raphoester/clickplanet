package shadowban_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
)

// stopAndFlush runs the banner and stops it, which flushes.
func stopAndFlush(banner *shadowban.Banner) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	banner.Run(ctx)
}

func TestBansSurviveARestart(t *testing.T) {
	clock := newClock()
	persistence := shadowban.NewMemoryPersistence()

	before := shadowban.New(config(), clock, persistence, failOnStateError(t))
	require.NoError(t, before.Load(t.Context()))

	before.Flag("repeat")
	clock.Advance(2 * time.Hour)
	before.Flag("repeat")
	clock.Advance(25 * time.Hour)
	before.Flag("repeat")
	before.Flag("fresh")
	stopAndFlush(before)

	after := shadowban.New(config(), clock, persistence, failOnStateError(t))
	require.NoError(t, after.Load(t.Context()))

	assert.True(t, after.Banned("repeat"))
	assert.True(t, after.Banned("fresh"))

	clock.Advance(2 * time.Hour)
	assert.True(t, after.Banned("repeat"), "a three-year ban outlives the restart")
	assert.False(t, after.Banned("fresh"), "the first offence still lapses on time")

	sentence, accepted := after.Flag("fresh")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence, "the offence count was kept")
}

func TestAFlushWritesOnlyTheScopesThatChanged(t *testing.T) {
	clock := newClock()
	persistence := shadowban.NewMemoryPersistence()
	banner := shadowban.New(config(), clock, persistence, failOnStateError(t))
	require.NoError(t, banner.Load(t.Context()))

	banner.Flag("first")
	require.NoError(t, banner.Flush(t.Context()))

	banner.Ban("second", time.Hour)
	require.NoError(t, banner.Flush(t.Context()))
	require.NoError(t, banner.Flush(t.Context()))

	saves := persistence.Saves()
	require.Len(t, saves, 2, "a flush with nothing changed writes nothing")
	assert.Equal(t, []shadowban.Record{{Scope: "second", Offences: 1, Until: clock.Now().Add(time.Hour)}}, saves[1])
}

func TestAFailedFlushKeepsTheBansForTheNextOne(t *testing.T) {
	persistence := shadowban.NewMemoryPersistence()
	banner := shadowban.New(config(), newClock(), persistence, failOnStateError(t))
	require.NoError(t, banner.Load(t.Context()))

	banner.Flag("bot")
	persistence.FailWith(errors.New("postgres is down"))
	require.Error(t, banner.Flush(t.Context()))

	persistence.Heal()
	require.NoError(t, banner.Flush(t.Context()))

	assert.Contains(t, persistence.Stored(), "bot")
}

func TestAFailedLoadRefusesTheBoot(t *testing.T) {
	persistence := shadowban.NewMemoryPersistence()
	persistence.FailWith(errors.New("postgres is down"))

	banner := shadowban.New(config(), newClock(), persistence, failOnStateError(t))

	require.Error(t, banner.Load(t.Context()))
}

func TestOneScopeIsUnbannedByDeletingItsRow(t *testing.T) {
	clock := newClock()
	until := clock.Now().Add(time.Hour)
	persistence := shadowban.NewMemoryPersistence(shadowban.Record{Scope: "keep", Flags: 1, Offences: 1, Until: until})

	banner := shadowban.New(config(), clock, persistence, failOnStateError(t))
	require.NoError(t, banner.Load(t.Context()))

	assert.True(t, banner.Banned("keep"))
	assert.False(t, banner.Banned("release"))
}

const legacyBans = `{"scope":"1.2.3.4","flags":3,"offences":2,"until":"2026-09-12T12:00:00Z"}
{"scope":"2a00:8c40:f0c5:6713::/64","flags":1,"offences":1,"until":"2026-09-11T13:00:00Z"}
`

func legacyConfig(t *testing.T, content string) shadowban.Config {
	t.Helper()
	c := config()
	c.LegacyStatePath = filepath.Join(t.TempDir(), "bans.jsonl")
	require.NoError(t, os.WriteFile(c.LegacyStatePath, []byte(content), 0o600))
	return c
}

func TestTheLegacyFileIsImportedIntoAnEmptyTableAndRenamedAfterTheFirstFlush(t *testing.T) {
	c := legacyConfig(t, legacyBans)
	persistence := shadowban.NewMemoryPersistence()

	banner := shadowban.New(c, newClock(), persistence, failOnStateError(t))
	require.NoError(t, banner.Load(t.Context()))

	sentence, banned := banner.Sentence("1.2.3.4")
	require.True(t, banned)
	assert.Equal(t, shadowban.Sentence{Flags: 3, Offence: 2, Until: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}, sentence)
	assert.FileExists(t, c.LegacyStatePath, "kept until postgres holds it")

	require.NoError(t, banner.Flush(t.Context()))

	assert.Len(t, persistence.Stored(), 2)
	assert.NoFileExists(t, c.LegacyStatePath)
	assert.FileExists(t, c.LegacyStatePath+".imported")
}

func TestACrashBeforeTheFirstFlushImportsAgain(t *testing.T) {
	c := legacyConfig(t, legacyBans)
	persistence := shadowban.NewMemoryPersistence()

	crashed := shadowban.New(c, newClock(), persistence, failOnStateError(t))
	require.NoError(t, crashed.Load(t.Context()))
	persistence.FailWith(errors.New("postgres is down"))
	require.Error(t, crashed.Flush(t.Context()))
	assert.FileExists(t, c.LegacyStatePath)

	persistence.Heal()
	rebooted := shadowban.New(c, newClock(), persistence, failOnStateError(t))
	require.NoError(t, rebooted.Load(t.Context()))

	assert.True(t, rebooted.Banned("1.2.3.4"))
}

func TestACorruptLegacyFileRefusesTheBoot(t *testing.T) {
	c := legacyConfig(t, "{not json\n")

	banner := shadowban.New(c, newClock(), shadowban.NewMemoryPersistence(), failOnStateError(t))

	require.ErrorContains(t, banner.Load(t.Context()), "line 1")
	assert.FileExists(t, c.LegacyStatePath)
}

func TestTheLegacyFileIsIgnoredOnceTheTableHoldsBans(t *testing.T) {
	c := legacyConfig(t, legacyBans)
	persistence := shadowban.NewMemoryPersistence(shadowban.Record{Scope: "stored", Offences: 1, Until: newClock().Now().Add(time.Hour)})

	var reported []error
	banner := shadowban.New(c, newClock(), persistence, func(err error) { reported = append(reported, err) })
	require.NoError(t, banner.Load(t.Context()))

	assert.True(t, banner.Banned("stored"))
	assert.False(t, banner.Banned("1.2.3.4"))
	require.Len(t, reported, 1)
	assert.ErrorContains(t, reported[0], "ignoring it")
}

func TestAnEmptyLegacyFileIsRenamedAtOnce(t *testing.T) {
	c := legacyConfig(t, "")

	banner := shadowban.New(c, newClock(), shadowban.NewMemoryPersistence(), failOnStateError(t))
	require.NoError(t, banner.Load(t.Context()))

	assert.FileExists(t, c.LegacyStatePath+".imported")
}
