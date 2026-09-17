package inmemory_visit_storage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func account(n int) players.AccountID {
	return players.AccountID{14: byte(n >> 8), 15: byte(n)}
}

func visit(n int, tag players.Tag, at time.Time) presence.Visit {
	return presence.Visit{Account: account(n), Tag: tag, Country: "fr", At: at}
}

func accounts(visits []presence.Visit) []players.AccountID {
	ids := make([]players.AccountID, 0, len(visits))
	for _, v := range visits {
		ids = append(ids, v.Account)
	}
	return ids
}

func TestALaterVisitReplacesTheAccountsLastOne(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))

	storage.Record(visit(1, "aaaaaa", start))
	later := visit(1, "bbbbbb", start.Add(time.Second))
	later.Country = "de"
	storage.Record(later)

	assert.Equal(t, []presence.Visit{later}, storage.Visits())
}

func TestANewAccountPushesOutTheOldestVisitOfAFullTag(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	for n := range inmemory_visit_storage.MaxVisitsPerTag {
		storage.Record(visit(n, "aaaaaa", start.Add(time.Duration(n)*time.Second)))
	}
	storage.Record(visit(100, "bbbbbb", start.Add(-time.Hour)))

	storage.Record(visit(inmemory_visit_storage.MaxVisitsPerTag, "aaaaaa", start.Add(time.Minute)))

	held := accounts(storage.Visits())
	assert.Len(t, held, inmemory_visit_storage.MaxVisitsPerTag+1)
	assert.NotContains(t, held, account(0), "the oldest of the full tag goes")
	assert.Contains(t, held, account(100), "another tag keeps its visits, however old")
	assert.Contains(t, held, account(inmemory_visit_storage.MaxVisitsPerTag))
}

func TestAnAccountAlreadyOnAFullTagPushesOutNobody(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	for n := range inmemory_visit_storage.MaxVisitsPerTag {
		storage.Record(visit(n, "aaaaaa", start))
	}

	storage.Record(visit(3, "aaaaaa", start.Add(time.Minute)))

	assert.Len(t, storage.Visits(), inmemory_visit_storage.MaxVisitsPerTag)
}

func TestAFullRosterRecordsNoNewAccountButKeepsTheOnesOnIt(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	for n := range inmemory_visit_storage.MaxVisits {
		storage.Record(visit(n, players.Tag(fmt.Sprintf("%06d", n)), start))
	}

	storage.Record(visit(inmemory_visit_storage.MaxVisits, "newtag", start))
	storage.Record(visit(0, "000000", start.Add(time.Minute)))

	held := storage.Visits()
	assert.Len(t, held, inmemory_visit_storage.MaxVisits)
	assert.NotContains(t, accounts(held), account(inmemory_visit_storage.MaxVisits))
	assert.Contains(t, held, visit(0, "000000", start.Add(time.Minute)))
}

func TestPruneForgetsOnlyTheStaleVisits(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	storage := inmemory_visit_storage.New(clock)
	storage.Record(visit(1, "aaaaaa", start))
	storage.Record(visit(2, "bbbbbb", start.Add(time.Minute)))

	clock.Advance(presence.TTL)
	storage.Prune()

	assert.Equal(t, []players.AccountID{account(2)}, accounts(storage.Visits()))
}
