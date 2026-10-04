package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var authoredAt = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func TestAnAuthorShowsItsStreakAsOfToday(t *testing.T) {
	today := players.DayOf(authoredAt)
	yesterday := players.DayOf(authoredAt.AddDate(0, 0, -1))
	before := players.DayOf(authoredAt.AddDate(0, 0, -2))

	alive := players.NamedAuthor(players.NewProfile(players.AccountID{15: 1}, "Ada", authoredAt), players.StreakOf(3, yesterday)).Shown(today)
	lapsed := players.NamedAuthor(players.NewProfile(players.AccountID{15: 2}, "Bob", authoredAt), players.StreakOf(3, before)).Shown(today)

	assert.Equal(t, uint32(3), alive.Streak().Days())
	assert.Equal(t, uint32(0), lapsed.Streak().Days())
}

func TestAGuestAuthorShowsNoStreak(t *testing.T) {
	today := players.DayOf(authoredAt)

	shown := players.GuestAuthor("91aa3d", players.StreakOf(9, today)).Shown(today)

	assert.Equal(t, players.Streak{}, shown.Streak())
	assert.Equal(t, "guest_91aa3d", shown.Name())
	assert.True(t, shown.Guest())
}

func TestANamedAuthorIsItsProfile(t *testing.T) {
	author := players.NamedAuthor(players.ProfileOf(players.AccountID{15: 1}, "Ada", authoredAt, true, 3), players.Streak{})

	assert.Equal(t, "Ada", author.Name())
	assert.False(t, author.Guest())
	assert.True(t, author.Admin())
	assert.Equal(t, players.Color(3), author.Color())
}

func TestARenamedAuthorKeepsItsMarkColorAndStreakAndIsNoLongerAGuest(t *testing.T) {
	streak := players.StreakOf(4, players.DayOf(authoredAt))
	named := players.NamedAuthor(players.ProfileOf(players.AccountID{15: 1}, "Ada", authoredAt, true, 3), streak)

	renamed := named.Renamed("Ada_L")

	assert.Equal(t, "Ada_L", renamed.Name())
	assert.True(t, renamed.Admin())
	assert.Equal(t, players.Color(3), renamed.Color())
	assert.Equal(t, streak, renamed.Streak())
	assert.False(t, players.GuestAuthor("91aa3d", players.Streak{}).Renamed("Ada").Guest())
}
