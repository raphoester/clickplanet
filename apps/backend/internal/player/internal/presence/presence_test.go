package presence_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func named(name players.Name, admin bool) wearing.Author {
	profile := players.ProfileOf(players.AccountID{}, name, time.Time{}, admin, 0)
	return wearing.AuthorOf(players.NamedAuthor(profile, players.Streak{}), titles.Standing{})
}

func player(name players.Name) wearing.Author {
	return named(name, false)
}

func guest(code players.GuestCode) wearing.Author {
	return wearing.AuthorOf(players.GuestAuthor(code, players.Streak{}), titles.Standing{})
}

func visit(key presence.Key, author wearing.Author, country string, at time.Time) presence.Visit {
	return presence.NewVisit(players.AccountID{}, author, "", country, at).Keyed(key)
}

type line struct {
	Key     presence.Key
	Name    string
	Country string
	Guest   bool
	Admin   bool
	Color   players.Color
	Streak  uint32
	Title   titles.Standing
}

func lineOf(entry presence.Entry) line {
	return line{
		Key: entry.Key(), Name: entry.Name(), Country: entry.Country(), Guest: entry.Guest(), Admin: entry.Admin(),
		Color: entry.Color(), Streak: entry.Streak(), Title: entry.Title(),
	}
}

func linesOf(roster []presence.Entry) []line {
	lines := make([]line, 0, len(roster))
	for _, entry := range roster {
		lines = append(lines, lineOf(entry))
	}
	return lines
}

func TestAVisitIsFreshForTheTTL(t *testing.T) {
	visit := presence.NewVisit(players.AccountID{}, wearing.Author{}, "", "", now)

	assert.True(t, visit.Fresh(now.Add(presence.TTL-time.Second)))
	assert.False(t, visit.Fresh(now.Add(presence.TTL)))
}

func TestTheRosterNamesEachVisitAsTheChatDoesAndNeverShowsTheAddress(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		presence.NewVisit(players.AccountID{}, player("Ada_L"), "aaaaaa", "fr", now).Keyed("1"),
		presence.NewVisit(players.AccountID{}, guest("0b1c2d"), "bbbbbb", "de", now).Keyed("2"),
	}, now)

	assert.Equal(t, []line{
		{Key: "1", Name: "Ada_L", Country: "fr"},
		{Key: "2", Name: "guest_0b1c2d", Country: "de", Guest: true},
	}, linesOf(roster))
}

func TestOnlyAPlayerWithAUsernameShowsAsAnAdmin(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		visit("1", named("Ada_L", true), "", now),
		visit("2", guest("0b1c2d"), "", now),
	}, now)

	assert.Equal(t, []line{
		{Key: "1", Name: "Ada_L", Admin: true},
		{Key: "2", Name: "guest_0b1c2d", Guest: true},
	}, linesOf(roster))
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

func TestAStaleVisitIsNotOnTheRoster(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		visit("", player("Ada"), "", now.Add(-presence.TTL)),
		visit("", player("Bob"), "", now.Add(-time.Second)),
	}, now)

	assert.Equal(t, []string{"Bob"}, names(roster))
}

func names(roster []presence.Entry) []string {
	names := make([]string, 0, len(roster))
	for _, entry := range roster {
		names = append(names, entry.Name())
	}
	return names
}

func TestALineCarriesTheColorAndTheStreakItWasAnnouncedWith(t *testing.T) {
	ada := players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada_L", time.Time{}, false, 3), players.StreakOf(12, players.DayOf(now)))

	entry := presence.EntryOf(visit("1", wearing.AuthorOf(ada, titles.Standing{}), "fr", now))

	assert.Equal(t, line{Key: "1", Name: "Ada_L", Country: "fr", Color: 3, Streak: 12}, lineOf(entry))
}

func TestALineCarriesTheTitleItsPlayerWears(t *testing.T) {
	og, _ := titles.NewCatalog().StandingOf("og")

	entry := presence.EntryOf(visit("1", player("Ada_L").Wearing(og), "", now))

	assert.Equal(t, line{Key: "1", Name: "Ada_L", Title: og}, lineOf(entry))
}

func TestARenamedLineKeepsItsMarkColorStreakAndTitle(t *testing.T) {
	og, _ := titles.NewCatalog().StandingOf("og")
	ada := players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada", time.Time{}, true, 3), players.StreakOf(4, players.DayOf(now)))

	entry := presence.EntryOf(visit("1", wearing.AuthorOf(ada, og).Renamed("Ada_L"), "fr", now))

	assert.Equal(t, line{Key: "1", Name: "Ada_L", Country: "fr", Admin: true, Color: 3, Streak: 4, Title: og}, lineOf(entry))
}
