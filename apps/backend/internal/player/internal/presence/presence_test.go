package presence_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestAVisitIsFreshForTheTTL(t *testing.T) {
	visit := presence.Visit{At: now}

	assert.True(t, visit.Fresh(now.Add(presence.TTL-time.Second)))
	assert.False(t, visit.Fresh(now.Add(presence.TTL)))
}

func TestAGuestNameIsCleanedAsTheChatCleansIt(t *testing.T) {
	assert.Equal(t, "Bob the builder", presence.GuestNameOf("  Bob\tthe\x00 builder\n"))
	assert.Equal(t, strings.Repeat("é", 24), presence.GuestNameOf(strings.Repeat("é", 24)))
}

func TestAGuestNameTheChatWouldRefuseIsEmpty(t *testing.T) {
	for _, value := range []string{"", " \n", strings.Repeat("a", 25), "\xff"} {
		assert.Empty(t, presence.GuestNameOf(value), "%q", value)
	}
}

func TestTheRosterNamesEachVisitAsTheChatDoes(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Username: "Ada_L", Tag: "aaaaaa", Country: "fr", At: now},
		{GuestName: "Bob", Tag: "bbbbbb", Country: "de", At: now},
		{Tag: "cccccc", Country: "jp", At: now},
	}, now)

	assert.Equal(t, []presence.Entry{
		{Name: "Ada_L", Tag: "aaaaaa", Country: "fr"},
		{Name: "guest_Bob", Tag: "bbbbbb", Country: "de", Guest: true},
		{Name: "guest_cccccc", Tag: "cccccc", Country: "jp", Guest: true},
	}, roster)
}

func TestOnlyAPlayerWithAUsernameShowsAsAnAdmin(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Username: "Ada_L", Admin: true, Tag: "aaaaaa", At: now},
		{GuestName: "Bob", Admin: true, Tag: "bbbbbb", At: now},
	}, now)

	assert.Equal(t, []presence.Entry{
		{Name: "Ada_L", Tag: "aaaaaa", Admin: true},
		{Name: "guest_Bob", Tag: "bbbbbb", Guest: true},
	}, roster)
}

func TestPlayersComeFirstThenGuestsEachByNameIgnoringCase(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{GuestName: "zed", Tag: "000001", At: now},
		{Username: "bob", Tag: "000002", At: now},
		{GuestName: "Amy", Tag: "000003", At: now},
		{Username: "Ada", Tag: "000004", At: now},
		{GuestName: "amy", Tag: "000000", At: now},
	}, now)

	assert.Equal(t, []string{"Ada", "bob", "guest_Amy", "guest_amy", "guest_zed"}, names(roster))
}

func TestTwoGuestsOfOneNameAreOrderedByTag(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{GuestName: "Bob", Tag: "bbbbbb", At: now},
		{GuestName: "Bob", Tag: "aaaaaa", At: now},
	}, now)

	assert.Equal(t, []players.Tag{"aaaaaa", "bbbbbb"}, []players.Tag{roster[0].Tag, roster[1].Tag})
}

func TestAStaleVisitIsNotOnTheRoster(t *testing.T) {
	roster := presence.RosterOf([]presence.Visit{
		{Username: "Ada", At: now.Add(-presence.TTL)},
		{Username: "Bob", At: now.Add(-time.Second)},
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
