package presence_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func named(name players.Name, admin bool) players.Player {
	profile := players.ProfileSnapshot{Name: name, Admin: admin}.Restore()
	return players.NamedPlayer(profile)
}

func player(name players.Name) players.Player {
	return named(name, false)
}

func guest(code players.GuestCode) players.Player {
	return players.GuestPlayer(code)
}

func visit(key presence.Key, shown players.Player, country string, at time.Time) presence.Visit {
	return presence.NewVisit(players.AccountID{}, shown, "", country, at).Keyed(key)
}

func TestPlayersComeFirstThenGuestsEachByNameIgnoringCase(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		visit("1", guest("ffffff"), "", now),
		visit("2", player("bob"), "", now),
		visit("3", guest("0a0a0a"), "", now),
		visit("4", player("Ada"), "", now),
		visit("5", player("ADA_2"), "", now),
	}, now)

	assert.Equal(t, []string{"Ada", "ADA_2", "bob", "guest_0a0a0a", "guest_ffffff"}, names(roster))
}

func TestTwoLinesOfOneNameAreOrderedByKey(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		visit("b", guest("0b1c2d"), "", now),
		visit("a", guest("0b1c2d"), "", now),
	}, now)

	assert.Equal(t, []presence.Key{"a", "b"}, []presence.Key{roster[0].Key(), roster[1].Key()})
}

func names(roster []presence.Entry) []string {
	names := make([]string, 0, len(roster))
	for _, entry := range roster {
		names = append(names, entry.Name())
	}
	return names
}
