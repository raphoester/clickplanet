package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestAnAuthorShowsItsStreakAsOfToday(t *testing.T) {
	today := players.DayOf(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))
	yesterday := players.DayOf(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	before := players.DayOf(time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC))

	alive := players.Author{Name: "Ada", Streak: players.Streak{Days: 3, LastDay: yesterday}}.Shown(today)
	lapsed := players.Author{Name: "Bob", Streak: players.Streak{Days: 3, LastDay: before}}.Shown(today)

	assert.Equal(t, uint32(3), alive.Streak.Days)
	assert.Equal(t, uint32(0), lapsed.Streak.Days)
}

func TestAGuestAuthorShowsNoStreak(t *testing.T) {
	today := players.DayOf(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))

	shown := players.Author{Name: "guest_91aa3d", Guest: true, Streak: players.Streak{Days: 9, LastDay: today}}.Shown(today)

	assert.Equal(t, players.Streak{}, shown.Streak)
}
