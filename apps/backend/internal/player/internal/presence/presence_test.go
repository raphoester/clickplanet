package presence_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func player(name string) players.Author {
	return players.Author{Name: name}
}

func guest(code string) players.Author {
	return players.Author{Name: "guest_" + code, Guest: true}
}

func TestAVisitIsFreshForTheTTL(t *testing.T) {
	visit := presence.Visit{At: now}

	assert.True(t, visit.Fresh(now.Add(presence.TTL-time.Second)))
	assert.False(t, visit.Fresh(now.Add(presence.TTL)))
}

func TestTheRosterNamesEachVisitAsTheChatDoesAndNeverShowsTheAddress(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Key: "1", Author: player("Ada_L"), Tag: "aaaaaa", Country: "fr", At: now},
		{Key: "2", Author: guest("0b1c2d"), Tag: "bbbbbb", Country: "de", At: now},
	}, now)

	assert.Equal(t, []presence.Entry{
		{Key: "1", Name: "Ada_L", Country: "fr"},
		{Key: "2", Name: "guest_0b1c2d", Country: "de", Guest: true},
	}, roster)
}

func TestOnlyAPlayerWithAUsernameShowsAsAnAdmin(t *testing.T) {
	admin := player("Ada_L")
	admin.Admin = true
	guestAdmin := guest("0b1c2d")
	guestAdmin.Admin = true

	roster := presence.RosterOf([]presence.Visit{
		{Key: "1", Author: admin, At: now},
		{Key: "2", Author: guestAdmin, At: now},
	}, now)

	assert.Equal(t, []presence.Entry{
		{Key: "1", Name: "Ada_L", Admin: true},
		{Key: "2", Name: "guest_0b1c2d", Guest: true},
	}, roster)
}

func TestPlayersComeFirstThenGuestsEachByNameIgnoringCase(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Key: "1", Author: guest("ffffff"), At: now},
		{Key: "2", Author: player("bob"), At: now},
		{Key: "3", Author: guest("0a0a0a"), At: now},
		{Key: "4", Author: player("Ada"), At: now},
		{Key: "5", Author: player("ADA_2"), At: now},
	}, now)

	assert.Equal(t, []string{"Ada", "ADA_2", "bob", "guest_0a0a0a", "guest_ffffff"}, names(roster))
}

func TestTwoLinesOfOneNameAreOrderedByKey(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Key: "b", Author: guest("0b1c2d"), At: now},
		{Key: "a", Author: guest("0b1c2d"), At: now},
	}, now)

	assert.Equal(t, []presence.Key{"a", "b"}, []presence.Key{roster[0].Key, roster[1].Key})
}

func TestAStaleVisitIsNotOnTheRoster(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Author: player("Ada"), At: now.Add(-presence.TTL)},
		{Author: player("Bob"), At: now.Add(-time.Second)},
	}, now)

	assert.Equal(t, []string{"Bob"}, names(roster))
}

func names(roster []presence.Entry) []string {
	names := make([]string, 0, len(roster))
	for _, entry := range roster {
		names = append(names, entry.Name)
	}
	return names
}
