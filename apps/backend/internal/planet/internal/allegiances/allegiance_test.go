package allegiances_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

var at = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

const (
	open   = allegiances.Lock(false)
	locked = allegiances.Lock(true)
)

var world = allegiances.EveryFlagStands{}

func TestTheFirstFlagStartsNoWait(t *testing.T) {
	first, err := allegiances.Allegiance{}.Pledged("fr", at, open, world)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "fr"}, first.TakeSnapshot())
}

func TestTheSameFlagChangesNothing(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "fr", ChangedAt: at}.Restore()

	again, err := kept.Pledged("fr", at.Add(time.Minute), open, world)
	require.NoError(t, err)

	assert.Equal(t, kept.TakeSnapshot(), again.TakeSnapshot())
}

func TestTheFirstChangeIsFreeAndStartsTheWait(t *testing.T) {
	first := allegiances.Snapshot{Flag: "fr"}.Restore()

	changed, err := first.Pledged("de", at, open, world)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, changed.TakeSnapshot())
}

func TestAChangeWithinTheHourIsRefusedWithTheFlagAndUntilWhen(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	_, err := kept.Pledged("fr", at.Add(time.Hour-time.Nanosecond), open, world)

	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrChangedTooSoon}, refusal)
}

func TestAChangeAnHourLaterIsAllowedAndStartsTheWaitAgain(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	changed, err := kept.Pledged("fr", at.Add(time.Hour), open, world)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Hour)}, changed.TakeSnapshot())
}

func TestAReleasedFlagChangesAtOnceAndStartsTheWaitAgain(t *testing.T) {
	released := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore().Released()
	assert.Equal(t, allegiances.Snapshot{Flag: "de"}, released.TakeSnapshot())

	changed, err := released.Pledged("fr", at.Add(time.Minute), open, world)
	require.NoError(t, err)
	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Minute)}, changed.TakeSnapshot())

	_, err = changed.Pledged("de", at.Add(2*time.Minute), open, world)
	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
}

func TestAReleaseDoesNotLiftALock(t *testing.T) {
	released := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore().Released()

	_, err := released.Pledged("fr", at, locked, world)

	require.ErrorIs(t, err, allegiances.ErrLocked)
}

func TestALockRefusesAChangeLongAfterTheWait(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	_, err := kept.Pledged("fr", at.Add(24*time.Hour), locked, world)

	require.ErrorIs(t, err, allegiances.ErrLocked)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Locked: true, Reason: allegiances.ErrLocked}, refusal)
}

func TestALockRefusesTheFirstChangeToo(t *testing.T) {
	first := allegiances.Snapshot{Flag: "de"}.Restore()

	_, err := first.Pledged("fr", at, locked, world)

	require.ErrorIs(t, err, allegiances.ErrLocked)
}

func TestALockStillTakesAFirstFlagAndTheSameFlag(t *testing.T) {
	first, err := allegiances.Allegiance{}.Pledged("fr", at, locked, world)
	require.NoError(t, err)
	assert.Equal(t, allegiances.Snapshot{Flag: "fr"}, first.TakeSnapshot())

	again, err := first.Pledged("fr", at, locked, world)
	require.NoError(t, err)
	assert.Equal(t, first.TakeSnapshot(), again.TakeSnapshot())
}

func TestAnotherFlagPlayedUnderALockIsRefusedWithTheLock(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de"}.Restore()

	require.NoError(t, kept.PlayError("de", locked))

	var refusal allegiances.Refusal
	require.ErrorAs(t, kept.PlayError("fr", locked), &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Locked: true, Reason: allegiances.ErrOtherFlag}, refusal)
}

func TestTheWaitEndsAnHourAfterAChangeAndNeverAfterNone(t *testing.T) {
	assert.Equal(t, at.Add(time.Hour), allegiances.ChangedAt(at).ChangeableAt())
	assert.True(t, allegiances.ChangedAt(time.Time{}).ChangeableAt().IsZero())
}

func TestOnlyTheFlagHeldIsPlayed(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	require.NoError(t, kept.PlayError("de", open))

	err := kept.PlayError("fr", open)
	require.ErrorIs(t, err, allegiances.ErrOtherFlag)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrOtherFlag}, refusal)
}

func TestAnotherFlagIsNotPlayedEvenWithNoWaitRunning(t *testing.T) {
	first := allegiances.Snapshot{Flag: "de"}.Restore()

	require.ErrorIs(t, first.PlayError("fr", open), allegiances.ErrOtherFlag)
}

func TestNoFlagIsPlayedBeforeOneIsPledged(t *testing.T) {
	err := allegiances.Allegiance{}.PlayError("fr", open)

	require.ErrorIs(t, err, allegiances.ErrNoFlag)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Empty(t, refusal.Flag)
}

func TestInAnArenaAFlagThatDoesNotStandIsRefused(t *testing.T) {
	arena := allegiances.StandingFlags{"de", "it"}
	kept := allegiances.Snapshot{Flag: "de"}.Restore()

	_, err := kept.Pledged("fr", at, open, arena)
	require.ErrorIs(t, err, allegiances.ErrDown)

	_, err = allegiances.Allegiance{}.Pledged("gb-eng", at, open, arena)
	require.ErrorIs(t, err, allegiances.ErrDown, "a newcomer picks a flag still in")
}

func TestInAnArenaANewcomerPicksAFlagStillIn(t *testing.T) {
	first, err := allegiances.Allegiance{}.Pledged("de", at, locked, allegiances.StandingFlags{"de"})
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "de"}, first.TakeSnapshot())
}

func TestAnAccountWhoseFlagIsOutChangesAtOncePastTheLockAndTheWait(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "fr", ChangedAt: at}.Restore()

	changed, err := kept.Pledged("de", at.Add(time.Minute), locked, allegiances.StandingFlags{"de"})
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at.Add(time.Minute)}, changed.TakeSnapshot(),
		"it starts the wait as any change does")
}

func TestARegionFlagsAccountChangesAtOnceInAnArena(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "gb-eng"}.Restore()

	changed, err := kept.Pledged("gb", at, locked, allegiances.StandingFlags{"gb"})
	require.NoError(t, err)

	assert.Equal(t, "gb", changed.TakeSnapshot().Flag)
}

func TestAnAccountWhoseFlagStandsIsHeldByTheLockInAnArena(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "fr"}.Restore()

	_, err := kept.Pledged("de", at, locked, allegiances.StandingFlags{"fr", "de"})

	require.ErrorIs(t, err, allegiances.ErrLocked)
}

func TestTheSameFlagAgainChangesNothingEvenOut(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "fr", ChangedAt: at}.Restore()

	again, err := kept.Pledged("fr", at, locked, allegiances.StandingFlags{})
	require.NoError(t, err)

	assert.Equal(t, kept.TakeSnapshot(), again.TakeSnapshot())
}
