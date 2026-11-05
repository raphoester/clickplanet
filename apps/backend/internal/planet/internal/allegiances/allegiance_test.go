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

func TestTheFirstFlagStartsNoWait(t *testing.T) {
	first, err := allegiances.Allegiance{}.Pledged("fr", at, open)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "fr"}, first.TakeSnapshot())
}

func TestTheSameFlagChangesNothing(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "fr", ChangedAt: at}.Restore()

	again, err := kept.Pledged("fr", at.Add(time.Minute), open)
	require.NoError(t, err)

	assert.Equal(t, kept.TakeSnapshot(), again.TakeSnapshot())
}

func TestTheFirstChangeIsFreeAndStartsTheWait(t *testing.T) {
	first := allegiances.Snapshot{Flag: "fr"}.Restore()

	changed, err := first.Pledged("de", at, open)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, changed.TakeSnapshot())
}

func TestAChangeWithinTheHourIsRefusedWithTheFlagAndUntilWhen(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	_, err := kept.Pledged("fr", at.Add(time.Hour-time.Nanosecond), open)

	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrChangedTooSoon}, refusal)
}

func TestAChangeAnHourLaterIsAllowedAndStartsTheWaitAgain(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	changed, err := kept.Pledged("fr", at.Add(time.Hour), open)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Hour)}, changed.TakeSnapshot())
}

func TestAReleasedFlagChangesAtOnceAndStartsTheWaitAgain(t *testing.T) {
	released := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore().Released()
	assert.Equal(t, allegiances.Snapshot{Flag: "de"}, released.TakeSnapshot())

	changed, err := released.Pledged("fr", at.Add(time.Minute), open)
	require.NoError(t, err)
	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Minute)}, changed.TakeSnapshot())

	_, err = changed.Pledged("de", at.Add(2*time.Minute), open)
	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
}

func TestAReleaseDoesNotLiftALock(t *testing.T) {
	released := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore().Released()

	_, err := released.Pledged("fr", at, locked)

	require.ErrorIs(t, err, allegiances.ErrLocked)
}

func TestALockRefusesAChangeLongAfterTheWait(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	_, err := kept.Pledged("fr", at.Add(24*time.Hour), locked)

	require.ErrorIs(t, err, allegiances.ErrLocked)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Locked: true, Reason: allegiances.ErrLocked}, refusal)
}

func TestALockRefusesTheFirstChangeToo(t *testing.T) {
	first := allegiances.Snapshot{Flag: "de"}.Restore()

	_, err := first.Pledged("fr", at, locked)

	require.ErrorIs(t, err, allegiances.ErrLocked)
}

func TestALockStillTakesAFirstFlagAndTheSameFlag(t *testing.T) {
	first, err := allegiances.Allegiance{}.Pledged("fr", at, locked)
	require.NoError(t, err)
	assert.Equal(t, allegiances.Snapshot{Flag: "fr"}, first.TakeSnapshot())

	again, err := first.Pledged("fr", at, locked)
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
