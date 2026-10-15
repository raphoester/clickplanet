package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	chosenAt = time.Date(2026, 10, 15, 18, 30, 0, 0, time.UTC)
	day      = 24 * time.Hour
	ada      = players.AccountID{15: 1}
)

func chosen(t *testing.T) players.Profile {
	t.Helper()

	profile, err := players.Unnamed(ada).Renamed("Ada", chosenAt)
	require.NoError(t, err)
	return profile
}

func TestTheFirstNameAnAccountPicksIsNotLimited(t *testing.T) {
	profile, err := players.Unnamed(ada).Renamed("Ada", chosenAt)

	require.NoError(t, err)
	assert.Equal(t, players.ProfileSnapshot{Account: ada, Name: "Ada", UpdatedAt: chosenAt, ChosenAt: chosenAt},
		profile.TakeSnapshot())
}

func TestAGivenNameIsNotAChoiceAndChangesAtOnce(t *testing.T) {
	given := players.NewProfile(ada, "BraveFox42", chosenAt)

	profile, err := given.Renamed("Ada", chosenAt.Add(time.Second))

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.TakeSnapshot().Name)
}

func TestAChangeOnDay29IsRefusedAndSaysWhenOneIsAllowed(t *testing.T) {
	for _, at := range []time.Time{chosenAt.Add(time.Minute), chosenAt.Add(29 * day), chosenAt.Add(30*day - time.Nanosecond)} {
		_, err := chosen(t).Renamed("Bob", at)

		require.ErrorIs(t, err, players.ErrRenamedTooSoon, "%s", at)
		var refusal players.RenameRefusal
		require.ErrorAs(t, err, &refusal)
		assert.Equal(t, chosenAt.Add(30*day), refusal.Until)
	}
}

func TestAChangeOnDay30IsAllowedAndStartsTheWaitAgain(t *testing.T) {
	renamed, err := chosen(t).Renamed("Bob", chosenAt.Add(30*day))
	require.NoError(t, err)

	_, err = renamed.Renamed("Cyd", chosenAt.Add(59*day))

	var refusal players.RenameRefusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, chosenAt.Add(60*day), refusal.Until)
}

func TestARenameKeepsTheAdminAndTheColor(t *testing.T) {
	kept := players.ProfileSnapshot{Account: ada, Name: "Ada", UpdatedAt: chosenAt, Admin: true, Color: 3}.Restore()

	renamed, err := kept.Renamed("Ada_L", chosenAt.Add(time.Hour))

	require.NoError(t, err)
	assert.Equal(t, players.ProfileSnapshot{
		Account: ada, Name: "Ada_L", UpdatedAt: chosenAt.Add(time.Hour), ChosenAt: chosenAt.Add(time.Hour), Admin: true, Color: 3,
	}, renamed.TakeSnapshot())
}

func TestANameNeverChosenIsRenamableAtAnyTime(t *testing.T) {
	assert.True(t, players.NameChosenAt(time.Time{}).RenamableAt().IsZero())
	assert.Equal(t, chosenAt.Add(30*day), players.NameChosenAt(chosenAt).RenamableAt())
}
