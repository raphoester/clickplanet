package shadowban_test

import (
	"context"
	"errors"
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
