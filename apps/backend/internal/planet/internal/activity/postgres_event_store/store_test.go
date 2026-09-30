package postgres_event_store_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/postgres_event_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_event_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "activity", migrations.FS)
	s.store = postgres_event_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var start = time.Date(2026, 9, 30, 12, 0, 0, 123_456_000, time.UTC)

var player = cpsession.AccountID(uuid.MustParse("00000000-0000-4000-8000-000000000001"))

type stored struct {
	At       time.Time
	Kind     string
	Scope    string
	Account  *string
	SignedIn bool
	Data     *string
}

func (s *testSuite) rows() []stored {
	rows, err := s.db.QueryContext(s.T().Context(),
		`SELECT at, kind, scope, account::text, signed_in, data::text FROM events ORDER BY id`)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(rows.Close()) }()

	var out []stored
	for rows.Next() {
		var (
			row           stored
			account, data sql.Null[string]
		)
		s.Require().NoError(rows.Scan(&row.At, &row.Kind, &row.Scope, &account, &row.SignedIn, &data))
		row.At = row.At.UTC()
		row.Account, row.Data = ptr(account), ptr(data)
		out = append(out, row)
	}
	s.Require().NoError(rows.Err())

	return out
}

func ptr[T any](value sql.Null[T]) *T {
	if !value.Valid {
		return nil
	}
	return &value.V
}

func (s *testSuite) TestEachKindKeepsItsOwnDataAndNoneIsNull() {
	guest := activity.Caller{Scope: "2001:db8::/64", Account: player}
	linked := activity.Caller{Scope: "2001:db8::/64", Account: player, SignedIn: true}
	nobody := activity.Caller{Scope: "2001:db8:1::/64", Account: cpsession.NoAccount}

	s.Require().NoError(s.store.Save(s.T().Context(), []activity.Event{
		{At: start, Kind: activity.KindClick, Caller: linked, Tile: 42, Country: "bg", Outcome: activity.OutcomeThrottled},
		{At: start, Kind: activity.KindTake, Caller: guest, Tile: 42, Country: "bg", Held: "fr"},
		{At: start, Kind: activity.KindTake, Caller: guest, Tile: 43, Country: "bg"},
		{At: start, Kind: activity.KindMap, Caller: nobody, Start: 0, End: 300_000, OffMap: true, Outcome: activity.OutcomeAccepted},
		{At: start, Kind: activity.KindStream, Caller: nobody},
		{At: start, Kind: activity.KindBoxCaught, Caller: nobody, Delay: 1234567 * time.Microsecond},
		{At: start, Kind: activity.KindBoxForeign, Caller: nobody},
	}))

	account := uuid.UUID(player).String()
	want := []struct {
		row  stored
		data string
	}{
		{
			stored{At: start, Kind: "click", Scope: "2001:db8::/64", Account: &account, SignedIn: true},
			`{"tile": 42, "country": "bg", "outcome": "throttled"}`,
		},
		{stored{At: start, Kind: "take", Scope: "2001:db8::/64", Account: &account}, `{"tile": 42, "country": "bg", "held": "fr"}`},
		{stored{At: start, Kind: "take", Scope: "2001:db8::/64", Account: &account}, `{"tile": 43, "country": "bg"}`},
		{
			stored{At: start, Kind: "map", Scope: "2001:db8:1::/64"},
			`{"start": 0, "end": 300000, "off_map": true, "outcome": "accepted"}`,
		},
		{stored{At: start, Kind: "stream", Scope: "2001:db8:1::/64"}, ""},
		{stored{At: start, Kind: "box_caught", Scope: "2001:db8:1::/64"}, `{"delay_us": 1234567}`},
		{stored{At: start, Kind: "box_foreign", Scope: "2001:db8:1::/64"}, ""},
	}

	rows := s.rows()
	s.Require().Len(rows, len(want))
	for i, row := range rows {
		data := row.Data
		row.Data = nil
		s.Equal(want[i].row, row)

		if want[i].data == "" {
			s.Nil(data, "a %s has no data", row.Kind)
			continue
		}
		s.Require().NotNil(data, row.Kind)
		s.JSONEq(want[i].data, *data)
	}
}

func (s *testSuite) TestARefusedClickOnAnyTileIdIsKept() {
	s.Require().NoError(s.store.Save(s.T().Context(), []activity.Event{
		{
			At: start, Kind: activity.KindClick, Caller: activity.Caller{Scope: "2001:db8::/64"}, Tile: 4_294_967_295,
			Country: "", Outcome: activity.OutcomeInvalid,
		},
	}))

	var tile int64
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT (data->>'tile')::bigint FROM events`).Scan(&tile))
	s.Equal(int64(4_294_967_295), tile)
}

func (s *testSuite) save(times ...time.Time) {
	events := make([]activity.Event, 0, len(times))
	for _, at := range times {
		events = append(events, activity.Event{At: at, Kind: activity.KindStream, Caller: activity.Caller{Scope: "2001:db8::/64"}})
	}
	s.Require().NoError(s.store.Save(s.T().Context(), events))
}

func (s *testSuite) times() []time.Time {
	rows := s.rows()
	out := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.At)
	}
	return out
}

func minutes(n ...int) []time.Time {
	out := make([]time.Time, 0, len(n))
	for _, m := range n {
		out = append(out, start.Add(time.Duration(m)*time.Minute))
	}
	return out
}

func (s *testSuite) TestDeleteBeforeDeletesTheOlderEventsAcrossChunks() {
	s.save(minutes(0, 1, 2, 3, 4, 5, 6)...)

	deleted, err := postgres_event_store.NewChunked(s.db, 2).DeleteBefore(s.T().Context(), start.Add(5*time.Minute))

	s.Require().NoError(err)
	s.Equal(int64(5), deleted)
	s.Equal(minutes(5, 6), s.times())
}

func (s *testSuite) TestDeleteBeforeWithNothingOlderDeletesNothing() {
	s.save(minutes(5)...)

	deleted, err := s.store.DeleteBefore(s.T().Context(), start)

	s.Require().NoError(err)
	s.Zero(deleted)
	s.Len(s.rows(), 1)
}

func (s *testSuite) TestDeleteOldestBeyondKeepsTheNewestAcrossChunks() {
	s.save(minutes(0, 1, 2)...)
	s.save(minutes(3, 4, 5, 6)...)

	deleted, err := postgres_event_store.NewChunked(s.db, 2).DeleteOldestBeyond(s.T().Context(), 2)

	s.Require().NoError(err)
	s.Equal(int64(5), deleted)
	s.Equal(minutes(5, 6), s.times())
}

func (s *testSuite) TestDeleteOldestBeyondUnderTheCapDeletesNothing() {
	s.save(minutes(0, 1)...)

	deleted, err := s.store.DeleteOldestBeyond(s.T().Context(), 2)
	s.Require().NoError(err)
	s.Zero(deleted)

	s.Require().NoError(s.db.Purge(s.T().Context()))
	deleted, err = s.store.DeleteOldestBeyond(s.T().Context(), 2)
	s.Require().NoError(err)
	s.Zero(deleted, "an empty table has no newest row")
}
