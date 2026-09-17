package get_roster_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/get_roster_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheRosterIsTheFreshVisitsAsOfTheClock(t *testing.T) {
	start := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	clock := cptime.NewFixedClock(start)
	visits := inmemory_visit_storage.New(clock)
	visits.Record(presence.Visit{Account: players.AccountID{15: 1}, Username: "Ada", Tag: "aaaaaa", Country: "fr", At: start})
	visits.Record(presence.Visit{Account: players.AccountID{15: 2}, Username: "Bob", Tag: "bbbbbb", Country: "de", At: start.Add(time.Minute)})

	clock.Advance(presence.TTL)

	assert.Equal(t, []presence.Entry{{Key: "2", Name: "Bob", Tag: "bbbbbb", Country: "de"}},
		get_roster_usecase.New(visits, clock).Execute())
}
