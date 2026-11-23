package world_replay_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/postgres_era_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/world_replay"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestRunErasuresSuite(t *testing.T) {
	suite.Run(t, new(erasuresSuite))
}

type erasuresSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_ledger_store.Store
}

const vandal = "7a1d3c55-0e4b-4c8f-9d2a-6b5e4f3a2c10"

var painted = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

type noCarries struct{}

func (noCarries) Carry(eras.Board, eras.Board) (ledger.Carry, error) { return noCarry{}, nil }

type noCarry struct{}

func (noCarry) Each(func(child, parent uint32)) {}

func (s *erasuresSuite) SetupSuite() {
	s.db = server.CreateDatabase(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *erasuresSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(eras.Reseed(s.T().Context(), s.db))
}

func (s *erasuresSuite) erasures() *world_replay.Erasures {
	world := world_replay.NewPostgresReplay(s.db, postgres_era_store.New(s.db), noCarries{})
	return world_replay.NewErasures(world, forgetting.NewPostgresQuery(s.db), cptime.NewFixedClock(painted.Add(100*time.Hour)))
}

// record keeps the events from position 1 on, and forgets the vandal's.
func (s *erasuresSuite) record(events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		entry.Stage = clicks.WorldStage
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i + 1), Entry: entry}) //nolint:gosec // a few events.
	}
	s.forget(vandal, ledger.Position(len(events)+1), slices.Values(stored)) //nolint:gosec // a few events.
}

func (s *erasuresSuite) forget(account string, before ledger.Position, entries func(func(inmemory_ledger_storage.Stored) bool)) {
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{
		From: before, Entries: entries,
		Marks: inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{{Account: account}: before}},
	}))
}

func (s *erasuresSuite) exec(statement string, args ...any) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), statement, args...)
	s.Require().NoError(err)
}

func take(account string, tile uint32, previous, country string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: "203.0.113.7", Account: account, Country: country, Previous: previous, At: at}
}

// previousSeen is what a take of the tile from the paint is read as taking it from, through the erasure.
func (s *erasuresSuite) previousSeen(erasure *ledger.Erasure, tile uint32) string {
	entry, err := take("", tile, "cn", "fr", painted.Add(90*time.Hour)).Entry()
	s.Require().NoError(err)
	entry.Stage = clicks.WorldStage
	event, kept, err := erasure.Read(entry)
	s.Require().NoError(err)
	s.Require().True(kept)
	var previous string
	event.Replay(func(taking ledger.Taking) { previous = taking.Previous })
	return previous
}

func (s *erasuresSuite) TestTheErasureBeforeAPositionSeesThroughWhatARevertForgotBeforeIt() {
	s.record(
		take(vandal, 1, "ie", "cn", painted),
		take(vandal, 2, "ie", "cn", painted),
		take(vandal, 3, "ie", "cn", painted.Add(time.Hour)),
	)

	erasure, err := s.erasures().Before(s.T().Context(), 3)
	s.Require().NoError(err)

	s.Equal("ie", s.previousSeen(erasure, 1))
	s.Equal("ie", s.previousSeen(erasure, 2))
	s.Equal("cn", s.previousSeen(erasure, 3), "the event at the position is the rewind's to read")
}

func (s *erasuresSuite) TestTheErasureMovesOnWithThePositionAndReadsNoEventBeforeItAgain() {
	s.record(
		take(vandal, 1, "ie", "cn", painted),
		take(vandal, 2, "ie", "cn", painted.Add(time.Hour)),
		take(vandal, 3, "ie", "cn", painted.Add(2*time.Hour)),
	)
	erasures := s.erasures()
	_, err := erasures.Before(s.T().Context(), 2)
	s.Require().NoError(err)
	// Read again, it would fail.
	s.exec(`UPDATE ledger_events SET kind = 'spread', payload = '{"taken":"not a list"}' WHERE position = 1`)

	moved, err := erasures.Before(s.T().Context(), 4)
	s.Require().NoError(err)
	s.Equal("ie", s.previousSeen(moved, 1))
	s.Equal("ie", s.previousSeen(moved, 3))

	_, err = erasures.Before(s.T().Context(), 3)
	s.Require().Error(err, "a position behind the one kept is read from the blank map again")
}

func (s *erasuresSuite) TestARevertReadsTheErasureFromTheBlankMapAgain() {
	s.record(take(ada, 1, "ie", "cn", painted))
	erasures := s.erasures()
	drawn, err := erasures.Before(s.T().Context(), 2)
	s.Require().NoError(err)
	s.Equal("cn", s.previousSeen(drawn, 1))

	s.forget(ada, 2, slices.Values([]inmemory_ledger_storage.Stored{}))
	erased, err := erasures.Before(s.T().Context(), 2)
	s.Require().NoError(err)

	s.Equal("ie", s.previousSeen(erased, 1))
}

func (s *erasuresSuite) TestAResetOrASwitchOfBoardEmptiesTheErasure() {
	for name, change := range map[string]func(at time.Time){
		"reset": func(at time.Time) { s.exec(`INSERT INTO map_resets (id, at) VALUES ('season-2', $1)`, at) },
		"switch": func(at time.Time) {
			s.Require().NoError(postgres_era_store.New(s.db).Start(s.T().Context(), clicks.World.On(eras.Earth2.ID), at))
		},
	} {
		s.Run(name, func() {
			s.SetupTest()
			s.record(
				take(vandal, 1, "ie", "cn", painted),
				take(ada, 2, "ie", "fr", painted.Add(2*time.Hour)),
			)
			erasures := s.erasures()
			before, err := erasures.Before(s.T().Context(), 2)
			s.Require().NoError(err)
			s.Equal("ie", s.previousSeen(before, 1))

			change(painted.Add(time.Hour))
			moved, err := erasures.Before(s.T().Context(), 3)
			s.Require().NoError(err)
			s.Equal("cn", s.previousSeen(moved, 1), "the paint is gone, and the map with it")

			fresh, err := s.erasures().Before(s.T().Context(), 3)
			s.Require().NoError(err)
			s.Equal("cn", s.previousSeen(fresh, 1))
		})
	}
}

func (s *erasuresSuite) TestTheEventsBeforeTheHeadAreReadByThePrimaryKey() {
	s.exec(`INSERT INTO ledger_events (position, map, board, kind, tile, scope, country, previous, taken_at)
		SELECT n, 'world', 1, 'take'::ledger_event_kind, n % 260503 + 1, '203.0.113.7', 'fr', 'de', now() - (200000 - n) * interval '1 second'
		FROM generate_series(0, 199999) n`)
	s.exec(`VACUUM ANALYZE ledger_events`)

	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(), "EXPLAIN "+world_replay.ErasedEvents, 150000, 151000, time.Time{})
	s.Require().NoError(err)
	defer func() { s.Require().NoError(rows.Close()) }()
	var lines []string
	for rows.Next() {
		var line string
		s.Require().NoError(rows.Scan(&line))
		lines = append(lines, line)
	}
	s.Require().NoError(rows.Err())
	plan := strings.Join(lines, "\n")

	s.Contains(plan, "ledger_events_pkey", plan)
	s.NotContains(plan, "Seq Scan on ledger_events", plan)
	s.NotContains(plan, "Sort", plan)
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
