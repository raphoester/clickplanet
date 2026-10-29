package casts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func account(t *testing.T, last string) casts.AccountID {
	t.Helper()
	id, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a" + last)
	require.NoError(t, err)
	return id
}

func take(who casts.AccountID, tile uint32, country, previous string) casts.Take {
	return casts.Take{At: at, Tile: tile, Country: country, Previous: previous, Account: who}
}

func TestFightersAreNumberedByTilesTakenFromAnotherFlag(t *testing.T) {
	ada, bob, eve := account(t, "01"), account(t, "02"), account(t, "03")
	takes := []casts.Take{
		take(ada, 1, "fr", "de"),
		take(bob, 2, "de", "fr"), take(bob, 3, "de", "fr"),
		take(eve, 4, "it", ""), take(eve, 5, "it", ""), take(eve, 6, "it", "it"),
	}

	cast := casts.CastOf(casts.CastID{1}, at, casts.Window{Since: at.Add(-time.Hour), Until: at}, takes,
		map[casts.AccountID]string{ada: "Ada", bob: "Bob", eve: "Eve"})

	assert.Equal(t, []casts.FighterSnapshot{{Number: 1, Account: bob, Name: "Bob"}, {Number: 2, Account: ada, Name: "Ada"}},
		cast.TakeSnapshot().Fighters, "filling empty ground and a shield on its own tile are no fight")
}

func TestAPlayerWithNoNameToShowIsNoFighter(t *testing.T) {
	ada, bob := account(t, "01"), account(t, "02")
	takes := []casts.Take{take(ada, 1, "fr", "de"), take(bob, 2, "de", "fr"), take(bob, 3, "de", "fr")}

	cast := casts.CastOf(casts.CastID{1}, at, casts.Window{}, takes, map[casts.AccountID]string{ada: "Ada"})

	assert.Equal(t, []casts.FighterSnapshot{{Number: 1, Account: ada, Name: "Ada"}}, cast.TakeSnapshot().Fighters)
	assert.Equal(t, []casts.NumberedTake{{At: at, Tile: 1, Country: "fr", Previous: "de", Fighter: 1}}, cast.Numbered(takes))
}

func TestATieIsOrderedByName(t *testing.T) {
	ada, bob := account(t, "01"), account(t, "02")
	takes := []casts.Take{take(bob, 1, "fr", "de"), take(ada, 2, "de", "fr")}

	cast := casts.CastOf(casts.CastID{1}, at, casts.Window{}, takes, map[casts.AccountID]string{ada: "Zed", bob: "Amy"})

	assert.Equal(t, "Amy", cast.TakeSnapshot().Fighters[0].Name)
}

func TestNumbersNameTheirAccounts(t *testing.T) {
	ada, bob := account(t, "01"), account(t, "02")
	cast := casts.CastOf(casts.CastID{1}, at, casts.Window{}, []casts.Take{take(ada, 1, "fr", "de"), take(bob, 2, "de", "fr")},
		map[casts.AccountID]string{ada: "Ada", bob: "Bob"})

	accounts, err := cast.AccountsOf([]casts.Number{2, 1})
	require.NoError(t, err)
	assert.Equal(t, []casts.AccountID{bob, ada}, accounts)

	_, err = cast.AccountsOf([]casts.Number{3})
	assert.ErrorIs(t, err, casts.ErrUnknownFighter)
}

func TestAFighterWhoseAccountIsGoneNamesNoOne(t *testing.T) {
	ada := account(t, "01")
	cast := casts.CastSnapshot{ID: casts.CastID{1}, Fighters: []casts.FighterSnapshot{{Number: 1}, {Number: 2, Account: ada, Name: "Ada"}}}.Restore()

	accounts, err := cast.AccountsOf([]casts.Number{1, 2})
	require.NoError(t, err)
	assert.Equal(t, []casts.AccountID{ada}, accounts)
}

func TestAWindowIsAtMost72HoursAndAZeroUntilIsNow(t *testing.T) {
	window, err := casts.WindowOf(at.Add(-12*time.Hour), time.Time{}, at)
	require.NoError(t, err)
	assert.Equal(t, casts.Window{Since: at.Add(-12 * time.Hour), Until: at}, window)

	for name, since := range map[string]time.Time{"none": {}, "after until": at.Add(time.Minute), "73 hours": at.Add(-73 * time.Hour)} {
		_, err := casts.WindowOf(since, at, at)
		assert.ErrorIs(t, err, casts.ErrInvalidWindow, name)
	}
}

func TestACastIsForgottenAfterThreeDays(t *testing.T) {
	assert.Equal(t, at.Add(-72*time.Hour), casts.ForgottenBefore(at))
}
