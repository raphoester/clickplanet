package inmemory_visit_storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// withoutKeys is the visits as announced, before the storage keyed them.
func withoutKeys(visits []presence.Visit) []presence.Visit {
	for i := range visits {
		visits[i].Key = ""
	}
	return visits
}

// changesOf is what a subscriber read so far, with nothing left waiting.
func changesOf(t *testing.T, changes <-chan presence.Change) []presence.Change {
	t.Helper()

	read := []presence.Change{}
	for {
		select {
		case change, open := <-changes:
			if !open {
				return read
			}
			read = append(read, change)
		default:
			return read
		}
	}
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

	assert.Equal(t, []presence.Visit{later}, withoutKeys(storage.Visits()))
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
	assert.Contains(t, withoutKeys(held), visit(0, "000000", start.Add(time.Minute)))
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

func TestAMoveCarriesTheVisitToTheNewAccountUnderItsName(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start))

	storage.Move(account(1), account(2), "Ada_L", false)

	moved := visit(2, "aaaaaa", start)
	moved.Username = "Ada_L"
	assert.Equal(t, []presence.Visit{moved}, withoutKeys(storage.Visits()), "one line, never the guest beside the player")
}

func TestAMoveToTheSameAccountRenamesIt(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start))

	storage.Move(account(1), account(1), "Ada_L", false)

	require.Len(t, storage.Visits(), 1)
	assert.Equal(t, players.Name("Ada_L"), storage.Visits()[0].Username)
}

func TestAMoveReplacesTheVisitTheAccountHeld(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start.Add(time.Second)))
	storage.Record(visit(2, "bbbbbb", start))

	storage.Move(account(1), account(2), "Ada_L", false)

	moved := visit(2, "aaaaaa", start.Add(time.Second))
	moved.Username = "Ada_L"
	assert.Equal(t, []presence.Visit{moved}, withoutKeys(storage.Visits()))
}

func TestAnAccountThatNeverAnnouncedMovesNothing(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))

	storage.Move(account(1), account(2), "Ada_L", false)
	storage.Rename(account(3), "Grace")

	assert.Empty(t, storage.Visits())
}

func TestARenameKeepsEverythingButTheName(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start))

	storage.Rename(account(1), "Ada_L")

	renamed := visit(1, "aaaaaa", start)
	renamed.Username = "Ada_L"
	assert.Equal(t, []presence.Visit{renamed}, withoutKeys(storage.Visits()))
}

func TestForgetTakesOnlyThatAccountOff(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start))
	storage.Record(visit(2, "aaaaaa", start))

	storage.Forget(account(1))
	storage.Forget(account(3))

	assert.Equal(t, []players.AccountID{account(2)}, accounts(storage.Visits()))
}

func TestEachNewAccountGetsAKeyThatItsLaterVisitsKeep(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start))
	storage.Record(visit(2, "aaaaaa", start))
	first := storage.Visits()

	storage.Record(visit(1, "bbbbbb", start.Add(time.Second)))
	storage.Move(account(1), account(3), "Ada_L", false)

	keys := map[players.AccountID]presence.Key{}
	for _, v := range first {
		keys[v.Account] = v.Key
	}
	assert.NotEqual(t, keys[account(1)], keys[account(2)])
	for _, v := range storage.Visits() {
		if v.Account == account(3) {
			assert.Equal(t, keys[account(1)], v.Key, "a visit keeps its key through a new announce and a sign-in")
		}
	}
}

func TestASubscriberReadsTheFreshRosterThenEveryChange(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	storage := inmemory_visit_storage.New(clock)
	storage.Record(visit(1, "aaaaaa", start))
	storage.Record(visit(2, "bbbbbb", start.Add(-presence.TTL)))

	roster, changes := storage.Subscribe(t.Context())
	require.Len(t, roster, 1, "a stale visit is not on the roster")
	one := roster[0]

	storage.Record(visit(1, "aaaaaa", start.Add(time.Second)))
	storage.Record(visit(3, "cccccc", start))
	storage.Rename(account(1), "Ada_L")
	storage.Forget(account(3))

	read := changesOf(t, changes)
	require.Len(t, read, 3, "an announce that changes nothing on the line is not a change")
	assert.False(t, read[0].Left)
	assert.Equal(t, "guest_cccccc", read[0].Entry.Name)
	assert.Equal(t, presence.Change{Entry: presence.Entry{Key: one.Key, Name: "Ada_L", Tag: "aaaaaa", Country: "fr"}}, read[1])
	assert.Equal(t, presence.Change{Entry: read[0].Entry, Left: true}, read[2])
}

func TestAStaleVisitThatAnnouncesAgainJoinsAgain(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	storage := inmemory_visit_storage.New(clock)
	storage.Record(visit(1, "aaaaaa", start))
	clock.Advance(presence.TTL)
	_, changes := storage.Subscribe(t.Context())

	storage.Record(visit(1, "aaaaaa", start.Add(presence.TTL)))

	read := changesOf(t, changes)
	require.Len(t, read, 1)
	assert.False(t, read[0].Left)
}

func TestAMoveOverAnotherVisitSaysThatOneLeft(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	storage.Record(visit(1, "aaaaaa", start))
	storage.Record(visit(2, "bbbbbb", start))
	roster, changes := storage.Subscribe(t.Context())
	require.Len(t, roster, 2)

	storage.Move(account(1), account(2), "Ada_L", false)

	read := changesOf(t, changes)
	require.Len(t, read, 2)
	assert.True(t, read[0].Left)
	assert.Equal(t, "guest_bbbbbb", read[0].Entry.Name)
	assert.Equal(t, "Ada_L", read[1].Entry.Name)
}

func TestPruneAndAFullTagSayWhoLeft(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	storage := inmemory_visit_storage.New(clock)
	for n := range inmemory_visit_storage.MaxVisitsPerTag {
		storage.Record(visit(n, "aaaaaa", start.Add(time.Duration(n)*time.Second)))
	}
	_, changes := storage.Subscribe(t.Context())

	storage.Record(visit(inmemory_visit_storage.MaxVisitsPerTag, "aaaaaa", start.Add(time.Minute)))
	clock.Advance(presence.TTL + time.Minute)
	storage.Prune()

	read := changesOf(t, changes)
	require.Len(t, read, 2+inmemory_visit_storage.MaxVisitsPerTag)
	assert.True(t, read[0].Left, "the oldest of the full tag leaves first")
	assert.False(t, read[1].Left)
	for _, change := range read[2:] {
		assert.True(t, change.Left)
	}
}

func TestASubscriberThatFallsBehindIsClosedAndTheOthersAreNot(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	_, slow := storage.Subscribe(t.Context())
	_, reading := storage.Subscribe(t.Context())

	for n := range inmemory_visit_storage.SubscriberBuffer + 1 {
		storage.Record(visit(n, players.Tag(fmt.Sprintf("%06d", n)), start))
		if n < inmemory_visit_storage.SubscriberBuffer {
			<-reading
		}
	}

	for range inmemory_visit_storage.SubscriberBuffer {
		<-slow
	}
	_, open := <-slow
	assert.False(t, open, "closed, so its stream starts over from a whole roster")
	change, open := <-reading
	assert.True(t, open)
	assert.False(t, change.Left)
}

func TestASubscriberIsClosedWhenItsContextEnds(t *testing.T) {
	storage := inmemory_visit_storage.New(cptime.NewFixedClock(start))
	ctx, cancel := context.WithCancel(t.Context())
	_, changes := storage.Subscribe(ctx)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-changes:
			return !open
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	storage.Record(visit(1, "aaaaaa", start))
}
