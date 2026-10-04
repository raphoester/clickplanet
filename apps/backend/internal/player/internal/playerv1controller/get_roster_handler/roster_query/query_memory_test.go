package roster_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestTheRosterIsTheFreshVisitsAsOfTheClock(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	visits := inmemory_visit_storage.New(clock)
	visits.Record(presence.Visit{Account: players.AccountID{15: 1}, Author: wearing.Author{Author: players.Author{Name: "Ada"}}, Tag: "aaaaaa", Country: "fr", At: start})
	visits.Record(presence.Visit{Account: players.AccountID{15: 2}, Author: wearing.Author{Author: players.Author{Name: "Bob"}}, Tag: "bbbbbb", Country: "de", At: start.Add(time.Minute)})

	clock.Advance(presence.TTL)

	assert.True(t, proto.Equal(
		&playerv1.GetRosterResponse{Entries: []*playerv1.RosterEntry{{Key: "2", Name: "Bob", CountryId: "de"}}},
		roster_query.NewMemoryQuery(visits, clock).Roster(),
	))
}

func TestEachLineSaysWhoIsPlayingAndPlayersComeBeforeGuests(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	visits := inmemory_visit_storage.New(clock)
	og := titles.Standing{Title: titles.OG{}}
	visits.Record(presence.Visit{
		Account: players.AccountID{15: 1}, Tag: "aaaaaa", Country: "fr", At: start,
		Author: wearing.Author{Author: players.Author{Name: "guest_a1b2c3", Guest: true}},
	})
	visits.Record(presence.Visit{
		Account: players.AccountID{15: 2}, Tag: "bbbbbb", Country: "de", At: start,
		Author: wearing.Author{
			Author: players.Author{Name: "Zed", Admin: true, Color: players.Color(playerv1.NameColor_NAME_COLOR_TEAL), Streak: players.Streak{Days: 4}},
			Worn:   og,
		},
	})

	assert.True(t, proto.Equal(&playerv1.GetRosterResponse{Entries: []*playerv1.RosterEntry{
		{
			Key: "2", Name: "Zed", CountryId: "de", Admin: true, Color: playerv1.NameColor_NAME_COLOR_TEAL, Streak: 4,
			WornTitle: &playerv1.Title{Id: "og", Name: "OG"},
		},
		{Key: "1", Name: "guest_a1b2c3", CountryId: "fr", Guest: true},
	}}, roster_query.NewMemoryQuery(visits, clock).Roster()))
}
