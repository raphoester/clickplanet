package schedule_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
)

var now = finale.Add(-time.Hour)

func draft(set schedule.SetName, startsAt, endsAt time.Time, priority schedule.Priority) schedule.Draft {
	return schedule.Draft{Set: set, StartsAt: startsAt, EndsAt: endsAt, Priority: priority}
}

func TestAnEntryIsMadeNowAndPlaysItsDraft(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, seasonEnds, 1), now)

	require.NoError(t, err)
	assert.Equal(t, schedule.EntrySnapshot{
		ID: id(1), Set: "calm", StartsAt: finale, EndsAt: seasonEnds, Priority: 1, MadeAt: now,
	}, made.TakeSnapshot())
}

func TestADraftIsRefusedWithEveryReason(t *testing.T) {
	_, err := schedule.NewEntry(id(1), draft("Calm!", now, now.Add(-time.Minute), 0), now)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry)
	assert.Contains(t, err.Error(), `"Calm!" is no rule set's name`)
	assert.Contains(t, err.Error(), "priority 0 is the seasons' phases")
	assert.Contains(t, err.Error(), "not after now")
	assert.Contains(t, err.Error(), "not after its start")
}

func TestAnEntryWithNoEndIsRefused(t *testing.T) {
	_, err := schedule.NewEntry(id(1), draft("calm", finale, time.Time{}, 1), now)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry)
}

func TestAnEntryToComeIsEditedAndKeepsItsPlaceInATie(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, seasonEnds, 1), now)
	require.NoError(t, err)

	edited, err := made.Edited(draft("frozen", finale.Add(time.Minute), seasonEnds, 2), now.Add(time.Minute))

	require.NoError(t, err)
	assert.Equal(t, schedule.EntrySnapshot{
		ID: id(1), Set: "frozen", StartsAt: finale.Add(time.Minute), EndsAt: seasonEnds, Priority: 2, MadeAt: now,
	}, edited.TakeSnapshot())
}

func TestAnEntryThatStartedNeitherChangesNorGoes(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, seasonEnds, 1), now)
	require.NoError(t, err)

	require.NoError(t, made.Changeable(finale.Add(-time.Nanosecond)))
	require.ErrorIs(t, made.Changeable(finale), schedule.ErrStarted, "at its start")
	require.ErrorIs(t, made.Changeable(seasonEnds.Add(time.Hour)), schedule.ErrStarted, "and once it ended")
	_, err = made.Edited(draft("calm", seasonEnds, seasonEnds.Add(time.Hour), 1), finale.Add(time.Minute))
	require.ErrorIs(t, err, schedule.ErrStarted)
}

func TestAnEditIsCheckedAsANewEntry(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, seasonEnds, 1), now)
	require.NoError(t, err)

	_, err = made.Edited(draft("calm", now.Add(-time.Minute), seasonEnds, 1), now)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry, "it would start in the past")
}

func TestAPhaseNamesARuleSet(t *testing.T) {
	require.NoError(t, sets.Validate())

	err := schedule.PhaseSets{Days: "default", Finale: ""}.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `finale: "" is no rule set's name`)
	assert.Contains(t, err.Error(), `over: "" is no rule set's name`)
}

func TestAnIDThatIsNoUUIDIsNoEntry(t *testing.T) {
	_, err := schedule.ParseEntryID("nope")

	require.ErrorIs(t, err, schedule.ErrUnknownEntry)
}
