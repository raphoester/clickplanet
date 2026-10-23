package allegiances_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

var at = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

func TestTheFirstFlagStartsNoWait(t *testing.T) {
	first, err := allegiances.Allegiance{}.Pledged("fr", at)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "fr"}, first.TakeSnapshot())
}

func TestTheSameFlagChangesNothing(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "fr", ChangedAt: at}.Restore()

	again, err := kept.Pledged("fr", at.Add(time.Minute))
	require.NoError(t, err)

	assert.Equal(t, kept.TakeSnapshot(), again.TakeSnapshot())
}

func TestTheFirstChangeIsFreeAndStartsTheWait(t *testing.T) {
	first := allegiances.Snapshot{Flag: "fr"}.Restore()

	changed, err := first.Pledged("de", at)
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, changed.TakeSnapshot())
}

func TestAChangeWithinTheHourIsRefusedWithTheFlagAndUntilWhen(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	_, err := kept.Pledged("fr", at.Add(time.Hour-time.Nanosecond))

	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrChangedTooSoon}, refusal)
}

func TestAChangeAnHourLaterIsAllowedAndStartsTheWaitAgain(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	changed, err := kept.Pledged("fr", at.Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Hour)}, changed.TakeSnapshot())
}

func TestTheWaitEndsAnHourAfterAChangeAndNeverAfterNone(t *testing.T) {
	assert.Equal(t, at.Add(time.Hour), allegiances.ChangedAt(at).ChangeableAt())
	assert.True(t, allegiances.ChangedAt(time.Time{}).ChangeableAt().IsZero())
}

func TestOnlyTheFlagHeldIsPlayed(t *testing.T) {
	kept := allegiances.Snapshot{Flag: "de", ChangedAt: at}.Restore()

	require.NoError(t, kept.PlayError("de"))

	err := kept.PlayError("fr")
	require.ErrorIs(t, err, allegiances.ErrOtherFlag)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, allegiances.Refusal{Flag: "de", Until: at.Add(time.Hour), Reason: allegiances.ErrOtherFlag}, refusal)
}

func TestAnotherFlagIsNotPlayedEvenWithNoWaitRunning(t *testing.T) {
	first := allegiances.Snapshot{Flag: "de"}.Restore()

	require.ErrorIs(t, first.PlayError("fr"), allegiances.ErrOtherFlag)
}

func TestNoFlagIsPlayedBeforeOneIsPledged(t *testing.T) {
	err := allegiances.Allegiance{}.PlayError("fr")

	require.ErrorIs(t, err, allegiances.ErrNoFlag)
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Empty(t, refusal.Flag)
}
