package shadowban_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newClock() *cptime.FixedClock {
	return cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
}

const threeYears = 3 * 365 * 24 * time.Hour

func config() shadowban.Config {
	return shadowban.Config{
		Enforce:        true,
		BanDurations:   []time.Duration{time.Hour, 24 * time.Hour, threeYears},
		StrikeMemory:   30 * 24 * time.Hour,
		ReflagInterval: 5 * time.Minute,
		SweepInterval:  time.Minute,
	}
}

func TestAFirstOffenceBansForTheFirstStep(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock, nil)

	require.False(t, banner.Banned("bot"))

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 1, sentence.Flags)
	assert.Equal(t, 1, sentence.Offence)
	assert.Equal(t, clock.Now().Add(time.Hour), sentence.Until)

	clock.Advance(59 * time.Minute)
	assert.True(t, banner.Banned("bot"))

	clock.Advance(2 * time.Minute)
	assert.False(t, banner.Banned("bot"))
	assert.Equal(t, 0, banner.Flagged())
}

func TestEnforceOffBansNothingAndStillCounts(t *testing.T) {
	c := config()
	c.Enforce = false

	banner := shadowban.New(c, newClock(), nil)

	_, accepted := banner.Flag("bot")
	require.True(t, accepted)

	assert.False(t, banner.Banned("bot"))
	assert.Equal(t, 1, banner.Flagged())
	assert.False(t, banner.Enforcing())
}

func TestAFlagInsideTheReflagIntervalSaysNothingNew(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock, nil)

	_, accepted := banner.Flag("bot")
	require.True(t, accepted)

	clock.Advance(time.Minute)
	sentence, accepted := banner.Flag("bot")
	assert.False(t, accepted)
	assert.Equal(t, 1, sentence.Flags)

	clock.Advance(5 * time.Minute)
	sentence, accepted = banner.Flag("bot")
	assert.True(t, accepted)
	assert.Equal(t, 2, sentence.Flags)
}

func TestAReflagExtendsTheRunningBanWithoutANewOffence(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock, nil)

	banner.Flag("bot")

	for range 11 {
		clock.Advance(50 * time.Minute)
		sentence, accepted := banner.Flag("bot")
		require.True(t, accepted)
		require.Equal(t, 1, sentence.Offence, "a caller that never stops is one offence")
	}

	clock.Advance(30 * time.Minute)
	assert.True(t, banner.Banned("bot"))
}

func TestAnOffenceAfterALapsedBanClimbsTheLadder(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock, nil)

	banner.Flag("bot")
	clock.Advance(2 * time.Hour)
	require.False(t, banner.Banned("bot"))

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence)
	assert.Equal(t, 2, sentence.Flags)
	assert.Equal(t, clock.Now().Add(24*time.Hour), sentence.Until)

	clock.Advance(23 * time.Hour)
	assert.True(t, banner.Banned("bot"))
}

func TestTheThirdOffenceBansForThreeYears(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock, nil)

	banner.Flag("bot")
	clock.Advance(2 * time.Hour)
	banner.Flag("bot")
	clock.Advance(25 * time.Hour)

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 3, sentence.Offence)
	assert.Equal(t, clock.Now().Add(threeYears), sentence.Until)

	clock.Advance(threeYears - time.Hour)
	assert.True(t, banner.Banned("bot"))

	clock.Advance(2 * time.Hour)
	assert.False(t, banner.Banned("bot"))
}

func TestTheLastStepRepeats(t *testing.T) {
	c := config()
	c.BanDurations = []time.Duration{time.Hour, 24 * time.Hour}

	clock := newClock()
	banner := shadowban.New(c, clock, nil)

	var sentence shadowban.Sentence
	for range 5 {
		sentence, _ = banner.Flag("bot")
		clock.Advance(25 * time.Hour)
	}

	assert.Equal(t, 5, sentence.Offence)
	assert.Equal(t, clock.Now().Add(-time.Hour), sentence.Until)
}

func TestAnEmptyScopeIsNeverBanned(t *testing.T) {
	banner := shadowban.New(config(), newClock(), nil)

	_, accepted := banner.Flag("")
	assert.False(t, accepted)
	assert.Equal(t, 0, banner.Flagged())
}

func TestBansSurviveARestart(t *testing.T) {
	c := config()
	c.StatePath = filepath.Join(t.TempDir(), "bans.jsonl")

	clock := newClock()
	before := shadowban.New(c, clock, failOnStateError(t))

	before.Flag("repeat")
	clock.Advance(2 * time.Hour)
	before.Flag("repeat")
	clock.Advance(25 * time.Hour)
	before.Flag("repeat")
	before.Flag("fresh")
	stopAndSave(before)

	after := shadowban.New(c, clock, failOnStateError(t))

	assert.True(t, after.Banned("repeat"))
	assert.True(t, after.Banned("fresh"))

	clock.Advance(2 * time.Hour)
	assert.True(t, after.Banned("repeat"), "a three-year ban outlives the restart")
	assert.False(t, after.Banned("fresh"), "the first offence still lapses on time")

	sentence, accepted := after.Flag("fresh")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence, "the offence count was kept")
}

func TestAnUnreadableStateIsReportedAndStartsEmpty(t *testing.T) {
	c := config()
	c.StatePath = filepath.Join(t.TempDir(), "bans.jsonl")
	require.NoError(t, os.WriteFile(c.StatePath, []byte("{not json\n"), 0o600))

	var reported error
	banner := shadowban.New(c, newClock(), func(err error) { reported = err })

	require.Error(t, reported)
	assert.Equal(t, 0, banner.Flagged())
}

func TestAMissingStateIsAFirstBoot(t *testing.T) {
	c := config()
	c.StatePath = filepath.Join(t.TempDir(), "bans.jsonl")

	banner := shadowban.New(c, newClock(), failOnStateError(t))

	assert.Equal(t, 0, banner.Flagged())
}

func TestOneScopeIsUnbannedByRemovingItsLine(t *testing.T) {
	c := config()
	c.StatePath = filepath.Join(t.TempDir(), "bans.jsonl")

	clock := newClock()
	before := shadowban.New(c, clock, failOnStateError(t))
	before.Flag("keep")
	before.Flag("release")
	stopAndSave(before)

	data, err := os.ReadFile(c.StatePath)
	require.NoError(t, err)

	var kept []byte
	for line := range bytes.Lines(data) {
		if !bytes.Contains(line, []byte(`"scope":"release"`)) {
			kept = append(kept, line...)
		}
	}
	require.NoError(t, os.WriteFile(c.StatePath, kept, 0o600))

	after := shadowban.New(c, clock, failOnStateError(t))
	assert.True(t, after.Banned("keep"))
	assert.False(t, after.Banned("release"))
}

func failOnStateError(t *testing.T) func(error) {
	t.Helper()
	return func(err error) { t.Errorf("unexpected state error: %v", err) }
}

func stopAndSave(banner *shadowban.Banner) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	banner.Run(ctx)
}
