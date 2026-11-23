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

func TestAChangeWithinTheHourIsRefusedWithTheFlagAndUntilWhen(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	_, err := kept.Pledged("fr", at.Add(time.Hour-time.Nanosecond), open, world)

	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrChangedTooSoon}, refusal)
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

func TestOnlyTheFlagHeldIsPlayed(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	require.NoError(t, kept.PlayError("de", open))

	err := kept.PlayError("fr", open)
	require.ErrorIs(t, err, allegiances.ErrOtherFlag)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrOtherFlag}, refusal)
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

	assert.Equal(t, allegiances.Snapshot{Flag: "de"}, changed.TakeSnapshot(), "a change forced by the knock-out starts no wait")
}

func TestAnAccountWhoseFlagWasOutChangesBackOnceAtOnceAfterTheKnockOut(t *testing.T) {
	forced, err := allegiances.Snapshot{Flag: "fr", ChangedAt: at}.Restore().
		Pledged("de", at.Add(time.Minute), locked, allegiances.StandingFlags{"de"})
	require.NoError(t, err)
	_, err = forced.Pledged("it", at.Add(2*time.Minute), locked, allegiances.StandingFlags{"de", "it"})
	require.ErrorIs(t, err, allegiances.ErrLocked, "the lock still holds while the battle plays")

	back, err := forced.Pledged("fr", at.Add(3*time.Minute), open, world)
	require.NoError(t, err)
	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(3 * time.Minute)}, back.TakeSnapshot())

	_, err = back.Pledged("de", at.Add(4*time.Minute), open, world)
	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon, "the change back starts the wait")
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
