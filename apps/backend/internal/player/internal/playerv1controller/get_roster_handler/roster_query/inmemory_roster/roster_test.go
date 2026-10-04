package inmemory_roster_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query/inmemory_roster"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func named(name players.Name, admin bool, color players.Color, streak players.Streak) players.Author {
	return players.NamedAuthor(players.ProfileOf(players.AccountID{}, name, time.Time{}, admin, color), streak)
}

func assertLines(t *testing.T, want []*playerv1.RosterEntry, got []*playerv1.RosterEntry) {
	t.Helper()

	assert.True(t, proto.Equal(&playerv1.GetRosterResponse{Entries: want}, &playerv1.GetRosterResponse{Entries: got}), "got %v", got)
}

func TestTheRosterIsTheFreshVisitsAsOfTheClock(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	visits := inmemory_visit_storage.New(clock)
	visits.Record(presence.NewVisit(players.AccountID{15: 1}, wearing.AuthorOf(named("Ada", false, 0, players.Streak{}), titles.Standing{}), "aaaaaa", "fr", start))
	visits.Record(presence.NewVisit(players.AccountID{15: 2}, wearing.AuthorOf(named("Bob", false, 0, players.Streak{}), titles.Standing{}), "bbbbbb", "de", start.Add(time.Minute)))

	clock.Advance(presence.TTL)

	assertLines(t, []*playerv1.RosterEntry{{Key: "2", Name: "Bob", CountryId: "de"}}, inmemory_roster.New(visits, clock).Lines())
}

func TestEachLineSaysWhoIsPlayingAndPlayersComeBeforeGuests(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	visits := inmemory_visit_storage.New(clock)
	og, _ := titles.NewCatalog().StandingOf("og")
	zed := named("Zed", true, players.Color(playerv1.NameColor_NAME_COLOR_TEAL), players.StreakOf(4, players.Day{}))
	visits.Record(presence.NewVisit(players.AccountID{15: 1}, wearing.AuthorOf(players.GuestAuthor("a1b2c3", players.Streak{}), titles.Standing{}), "aaaaaa", "fr", start))
	visits.Record(presence.NewVisit(players.AccountID{15: 2}, wearing.AuthorOf(zed, og), "bbbbbb", "de", start))

	assertLines(t, []*playerv1.RosterEntry{
		{
			Key: "2", Name: "Zed", CountryId: "de", Admin: true, Color: playerv1.NameColor_NAME_COLOR_TEAL, Streak: 4,
			WornTitle: &playerv1.Title{Id: "og", Name: "OG"},
		},
		{Key: "1", Name: "guest_a1b2c3", CountryId: "fr", Guest: true},
	}, inmemory_roster.New(visits, clock).Lines())
}
