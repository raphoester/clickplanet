package changes_query_test

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler/changes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_ledger_store.Store
}

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
)

var at = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) record(events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i), Entry: entry}) //nolint:gosec // a few events.
	}
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{Entries: slices.Values(stored)}))
}

// Every page of the window, one after the other.
func (s *testSuite) ledger(since, until time.Time) []*planetv1.LedgerChange {
	query := changes_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at.Add(time.Hour)))
	var changes []*planetv1.LedgerChange
	for page := ""; ; {
		answer, err := query.Ledger(s.T().Context(), since, until, page)
		s.Require().NoError(err)
		changes = append(changes, answer.GetChanges()...)
		if page = answer.GetNextPage(); page == "" {
			return changes
		}
	}
}

func everyKind() []ledger.Event {
	return []ledger.Event{
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", Previous: "de", At: at},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at},
		ledger.Spreading{Tile: 3, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at, Impacts: []clicks.Impact{
			{Tile: 3, Owner: "es", Outcome: clicks.Taken},
			{Tile: 4, Owner: "de", Outcome: clicks.Taken},
			{Tile: 5, Owner: "it", Outcome: clicks.Shielded, Shields: 2},
			{Tile: 6, Outcome: clicks.Taken},
		}},
		ledger.Enclosing{Tile: 7, Scope: "203.0.113.7", Account: ada, Country: "it", At: at, Impacts: []clicks.Impact{
			{Tile: 7, Owner: "fr", Outcome: clicks.Taken},
			{Tile: 8, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Bombing{Scope: "203.0.113.7", Account: ada, At: at, Blast: clicks.Blast{
			Tile: 9, CountryID: "fr", Cleared: []uint32{10, 9}, Owners: []string{"pl", "de"}, Struck: []uint32{11}, Left: []int{0},
		}},
		ledger.Striking{Tile: 11, Scope: "203.0.113.7", Account: ada, Country: "fr", Owner: "de", Shields: 1, At: at},
		ledger.Shielding{Tile: 12, Scope: "203.0.113.7", Account: ada, Country: "fr", Shields: 3, At: at},
		ledger.Fortifying{
			Tile: 12, Landmass: 4, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at,
			Raised: []clicks.TileShields{{Tile: 12, Shields: 4}, {Tile: 2, Shields: 1}},
		},
		ledger.Taking{Tile: 13, Scope: "198.51.100.4", Account: bob, Country: "de", Previous: "fr", At: at},
		ledger.Taking{Tile: 14, Scope: "198.51.100.4", Country: "fr", Previous: "de", At: at},
	}
}

func change(by string, update *planetv1.TileUpdate) *planetv1.LedgerChange {
	account, ok := strings.CutPrefix(by, "account:")
	if !ok {
		account = ""
	}
	return &planetv1.LedgerChange{At: timestamppb.New(at), Update: update, By: by, AccountId: account}
}

func (s *testSuite) TestEveryKindGoesOutAsTheStreamShowedItWithWhoMadeIt() {
	s.record(everyKind()...)

	s.equal([]*planetv1.LedgerChange{
		change("account:"+ada, &planetv1.TileUpdate{TileId: 1, CountryId: "fr", PreviousCountryId: "de", Clicked: true}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 2, CountryId: "fr", Clicked: true}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 3, CountryId: "fr", PreviousCountryId: "es"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 4, CountryId: "fr", PreviousCountryId: "de"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 5, CountryId: "it", PreviousCountryId: "it", Shields: 2}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 6, CountryId: "fr"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 7, CountryId: "it", PreviousCountryId: "fr"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 8, CountryId: "it", PreviousCountryId: "fr"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 10, PreviousCountryId: "pl"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 9, PreviousCountryId: "de"}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 11, CountryId: "de", PreviousCountryId: "de", Clicked: true, Shields: 1}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 12, CountryId: "fr", PreviousCountryId: "fr", Shields: 3}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 12, CountryId: "fr", PreviousCountryId: "fr", Shields: 4}),
		change("account:"+ada, &planetv1.TileUpdate{TileId: 2, CountryId: "fr", PreviousCountryId: "fr", Shields: 1}),
		change("account:"+bob, &planetv1.TileUpdate{TileId: 13, CountryId: "de", PreviousCountryId: "fr", Clicked: true}),
		change("scope:198.51.100.4", &planetv1.TileUpdate{TileId: 14, CountryId: "fr", PreviousCountryId: "de", Clicked: true}),
	}, s.ledger(at, at.Add(time.Minute)))
}

func (s *testSuite) TestTheQueryAnswersWhatEventShowShowsForEveryKind() {
	events := everyKind()
	s.record(events...)

	kinds := make([]string, 0, len(events))
	var shown []*planetv1.LedgerChange
	for _, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		kinds = append(kinds, entry.Kind)
		by := "scope:" + entry.Scope
		if entry.Account != "" {
			by = "account:" + entry.Account
		}

		event.Show(func(scene ledger.Scene) {
			switch {
			case scene.Change != nil:
				shown = append(shown, change(by, wire(scene.Change.TileUpdate)))
			case scene.Blast != nil:
				for i, tile := range scene.Blast.Cleared {
					shown = append(shown, change(by, wire(clicks.TileUpdate{Tile: tile, Previous: scene.Blast.Owners[i]})))
				}
			case scene.Spread != nil, scene.Enclosure != nil:
			}
		})
	}

	s.Subset(kinds, ledger.Kinds(), "every kind the ledger keeps is in the test")
	s.equal(shown, s.ledger(at, at.Add(time.Minute)))
}

func wire(update clicks.TileUpdate) *planetv1.TileUpdate {
	return &planetv1.TileUpdate{
		TileId: update.Tile, CountryId: update.Value, PreviousCountryId: update.Previous, Clicked: update.Clicked,
		Shields: uint32(update.Shields), //nolint:gosec // a few shields.
	}
}

func (s *testSuite) TestOnlyTheWindowIsReadAndAnUnsetUntilIsNow() {
	s.record(
		ledger.Taking{Tile: 1, Scope: "a", Country: "fr", At: at.Add(-2 * time.Hour)},
		ledger.Taking{Tile: 2, Scope: "a", Country: "fr", At: at},
		ledger.Taking{Tile: 3, Scope: "a", Country: "fr", At: at.Add(30 * time.Minute)},
		ledger.Taking{Tile: 4, Scope: "a", Country: "fr", At: at.Add(2 * time.Hour)},
	)

	tiles := func(changes []*planetv1.LedgerChange) []uint32 {
		ids := make([]uint32, 0, len(changes))
		for _, change := range changes {
			ids = append(ids, change.GetUpdate().GetTileId())
		}
		return ids
	}
	s.Equal([]uint32{2}, tiles(s.ledger(at.Add(-time.Hour), at.Add(30*time.Minute))), "until is left out")
	s.Equal([]uint32{2, 3}, tiles(s.ledger(at.Add(-time.Hour), time.Time{})), "the clock reads an hour after at")
}

func (s *testSuite) TestAChangeBehindTheHeadIsNotRead() {
	s.record(
		ledger.Taking{Tile: 1, Scope: "a", Country: "fr", At: at},
		ledger.Taking{Tile: 2, Scope: "a", Country: "fr", At: at},
	)
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE ledger_head SET head = 1`)
	s.Require().NoError(err)

	s.Len(s.ledger(at, at.Add(time.Minute)), 1)
}

func (s *testSuite) TestADeletedAccountsChangesAreItsAddresss() {
	s.record(ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", At: at})
	id, err := ledger.AccountIDOf(ada)
	s.Require().NoError(err)

	s.Require().NoError(s.store.AnonymizeTakes(s.T().Context(), id))

	changes := s.ledger(at, at.Add(time.Minute))
	s.Equal("scope:203.0.113.7", changes[0].GetBy())
	s.Empty(changes[0].GetAccountId())
}

func (s *testSuite) TestAPageEndsOnAWholeEventAndTheNextOneStartsAfterIt() {
	events := make([]ledger.Event, 0, changes_query.PageEvents+1)
	for i := range changes_query.PageEvents - 1 {
		events = append(events, ledger.Taking{Tile: uint32(i%1000 + 1), Scope: "a", Country: "fr", At: at}) //nolint:gosec // under 1000.
	}
	events = append(events,
		ledger.Bombing{Scope: "a", At: at, Blast: clicks.Blast{Tile: 9, CountryID: "fr", Cleared: []uint32{10, 9}, Owners: []string{"pl", "de"}}},
		ledger.Taking{Tile: 5, Scope: "b", Country: "de", At: at},
	)
	s.record(events...)
	query := changes_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at.Add(time.Hour)))

	first, err := query.Ledger(s.T().Context(), at, at.Add(time.Minute), "")
	s.Require().NoError(err)
	second, err := query.Ledger(s.T().Context(), at, at.Add(time.Minute), first.GetNextPage())
	s.Require().NoError(err)

	changes := first.GetChanges()
	s.Require().Len(changes, changes_query.PageEvents+1, "the bomb's two tiles are on the page it ends")
	s.Equal(uint32(10), changes[changes_query.PageEvents-1].GetUpdate().GetTileId())
	s.Equal(uint32(9), changes[changes_query.PageEvents].GetUpdate().GetTileId())
	s.Equal(strconv.Itoa(changes_query.PageEvents), first.GetNextPage())
	s.Require().Len(second.GetChanges(), 1)
	s.Equal("scope:b", second.GetChanges()[0].GetBy())
	s.Empty(second.GetNextPage(), "the last page")
}

func (s *testSuite) TestAnEventWithNoChangeStillCountsOnItsPage() {
	events := make([]ledger.Event, 0, changes_query.PageEvents+1)
	for range changes_query.PageEvents {
		events = append(events, ledger.Bombing{Scope: "a", At: at, Blast: clicks.Blast{CountryID: "fr"}})
	}
	events = append(events, ledger.Taking{Tile: 5, Scope: "b", Country: "de", At: at})
	s.record(events...)

	changes := s.ledger(at, at.Add(time.Minute))

	s.Require().Len(changes, 1, "a bomb in the sea changes nothing")
	s.Equal("scope:b", changes[0].GetBy())
}

func (s *testSuite) TestAPageThisQueryNeverGaveIsRefused() {
	query := changes_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at))

	for _, page := range []string{"x", "-1", "1.5"} {
		_, err := query.Ledger(s.T().Context(), at.Add(-time.Hour), at, page)
		s.ErrorIs(err, changes_query.ErrInvalidPage, page)
	}
}

func (s *testSuite) TestAWindowOfNothingOrOfMoreThan72HoursIsRefused() {
	query := changes_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at))

	for _, window := range [][2]time.Time{
		{},
		{at, at},
		{at, at.Add(-time.Hour)},
		{at.Add(-72*time.Hour - time.Second), at},
	} {
		_, err := query.Ledger(s.T().Context(), window[0], window[1], "")
		s.ErrorIs(err, changes_query.ErrInvalidWindow, window)
	}
}

func (s *testSuite) TestAStoreFailureIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := changes_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at)).Ledger(ctx, at.Add(-time.Hour), at, "")

	s.Error(err)
}

func (s *testSuite) equal(expected, actual []*planetv1.LedgerChange) {
	s.T().Helper()
	s.Require().Len(actual, len(expected), "%v", actual)
	for i := range expected {
		s.True(proto.Equal(expected[i], actual[i]), "change %d: expected %v, got %v", i, expected[i], actual[i])
	}
}
