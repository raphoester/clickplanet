package schedule_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

var now = finale.Add(-time.Hour)

const words schedule.Description = "Triple points. A bonus box every minute."

func draft(set schedule.SetName, startsAt, endsAt time.Time, priority schedule.Priority) schedule.Draft {
	return schedule.Draft{
		Set: set, Kind: schedule.Battle, Description: words, StartsAt: startsAt, EndsAt: endsAt, Priority: priority, Times: 3,
	}
}

func TestAnEntryIsMadeNowAndPlaysItsDraft(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, battleEnds, 1), now)

	require.NoError(t, err)
	assert.Equal(t, schedule.EntrySnapshot{
		ID: id(1), Set: "calm", Kind: schedule.Battle, Description: words, StartsAt: finale, EndsAt: battleEnds, Priority: 1,
		Times: 3, MadeAt: now,
	}, made.TakeSnapshot())
}

func TestADraftIsRefusedWithEveryReason(t *testing.T) {
	long := draft("Calm!", now, now.Add(-time.Minute), 0)
	long.Description = schedule.Description(strings.Repeat("x", schedule.MaxDescription+1))

	_, err := schedule.NewEntry(id(1), long, now)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry)
	require.ErrorIs(t, err, schedule.ErrLongDescription)
	assert.Contains(t, err.Error(), `"Calm!" is no rule set's name`)
	assert.Contains(t, err.Error(), "the description is too long: 301 characters, at most 300")
	assert.Contains(t, err.Error(), "not after now")
	assert.Contains(t, err.Error(), "not after its start")
}

func TestADescriptionIsCountedInCharactersAndMayBeEmpty(t *testing.T) {
	for _, description := range []schedule.Description{"", schedule.Description(strings.Repeat("é", schedule.MaxDescription))} {
		fits := draft("calm", finale, battleEnds, 1)
		fits.Description = description

		_, err := schedule.NewEntry(id(1), fits, now)

		require.NoError(t, err)
	}
}

func TestAnEntryOfAKindNobodyKnowsIsRefused(t *testing.T) {
	for _, kind := range []schedule.Kind{"", "siege"} {
		unknown := draft("calm", finale, battleEnds, 1)
		unknown.Kind = kind

		_, err := schedule.NewEntry(id(1), unknown, now)

		require.ErrorIs(t, err, schedule.ErrUnknownKind, "%q", kind)
		assert.NotErrorIs(t, err, schedule.ErrInvalidEntry)
	}
}

func TestAPlainEntryIsScheduled(t *testing.T) {
	plain := draft("calm", finale, battleEnds, 1)
	plain.Kind, plain.Times = schedule.Plain, 0

	made, err := schedule.NewEntry(id(1), plain, now)

	require.NoError(t, err)
	assert.Equal(t, schedule.Plain, made.TakeSnapshot().Kind)
}

func TestAHalloweenEntryIsScheduled(t *testing.T) {
	halloween := draft("calm", finale, battleEnds, 1)
	halloween.Kind, halloween.Times = schedule.Halloween, 0

	made, err := schedule.NewEntry(id(1), halloween, now)

	require.NoError(t, err)
	assert.Equal(t, schedule.Halloween, made.TakeSnapshot().Kind)
}

func TestABattleScoresOneToTenTimesAndAPlainOrHalloweenEntryNone(t *testing.T) {
	for _, refused := range []struct {
		kind  schedule.Kind
		times schedule.Times
	}{{schedule.Battle, 0}, {schedule.Battle, 11}, {schedule.Plain, 1}, {schedule.Halloween, 1}} {
		asked := draft("calm", finale, battleEnds, 1)
		asked.Kind, asked.Times = refused.kind, refused.times

		_, err := schedule.NewEntry(id(1), asked, now)

		require.ErrorIs(t, err, schedule.ErrInvalidEntry, "%s × %d", refused.kind, refused.times)
	}
	for _, times := range []schedule.Times{1, 10} {
		asked := draft("calm", finale, battleEnds, 1)
		asked.Times = times

		_, err := schedule.NewEntry(id(1), asked, now)

		require.NoError(t, err)
	}
}

func TestAnyPriorityIsScheduled(t *testing.T) {
	_, err := schedule.NewEntry(id(1), draft("calm", finale, battleEnds, 0), now)

	require.NoError(t, err)
}

func TestAnEntryWithNoEndIsRefused(t *testing.T) {
	_, err := schedule.NewEntry(id(1), draft("calm", finale, time.Time{}, 1), now)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry)
}

func TestAnEntryToComeIsEditedAndKeepsItsPlaceInATie(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, battleEnds, 1), now)
	require.NoError(t, err)
	edit := draft("frozen", finale.Add(time.Minute), battleEnds, 2)
	edit.Kind, edit.Times = schedule.Plain, 0

	edited, err := made.Edited(edit, now.Add(time.Minute))

	require.NoError(t, err)
	assert.Equal(t, schedule.EntrySnapshot{
		ID: id(1), Set: "frozen", Kind: schedule.Plain, Description: words, StartsAt: finale.Add(time.Minute), EndsAt: battleEnds,
		Priority: 2, MadeAt: now,
	}, edited.TakeSnapshot())
}

func TestAnEntryThatStartedNeitherChangesNorGoes(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, battleEnds, 1), now)
	require.NoError(t, err)

	require.NoError(t, made.Changeable(finale.Add(-time.Nanosecond)))
	require.ErrorIs(t, made.Changeable(finale), schedule.ErrStarted, "at its start")
	require.ErrorIs(t, made.Changeable(battleEnds.Add(time.Hour)), schedule.ErrStarted, "and once it ended")
	_, err = made.Edited(draft("calm", battleEnds, battleEnds.Add(time.Hour), 1), finale.Add(time.Minute))
	require.ErrorIs(t, err, schedule.ErrStarted)
}

func TestAnEditIsCheckedAsANewEntry(t *testing.T) {
	made, err := schedule.NewEntry(id(1), draft("calm", finale, battleEnds, 1), now)
	require.NoError(t, err)

	_, err = made.Edited(draft("calm", now.Add(-time.Minute), battleEnds, 1), now)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry, "it would start in the past")
}

func TestAnIDThatIsNoUUIDIsNoEntry(t *testing.T) {
	_, err := schedule.ParseEntryID("nope")

	require.ErrorIs(t, err, schedule.ErrUnknownEntry)
}
