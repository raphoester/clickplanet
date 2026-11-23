package recent_history_query_test

import (
	"context"
	"encoding/binary"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/postgres_era_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/world_replay"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_recent_map_history_handler/recent_history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/postgres_reset_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_ledger_store.Store
	clock *cptime.FixedClock
	tiles *stubTiles
}

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5c0ffee0-1d2e-4f3a-8b4c-5d6e7f8a9b0c"
	eve = "e7e7e7e7-0000-4000-8000-000000000001"
)

var (
	until = time.Date(2026, 11, 11, 12, 30, 0, 0, time.UTC)
	since = until.Add(-24 * time.Hour)
	now   = until.Add(17*time.Minute + 13*time.Second)
)

type stubTiles struct {
	// The world's on earth-1 when unset.
	playing   clicks.Stage
	now, kept clicks.DenseBatch
	reads     atomic.Int32
	entered   chan struct{}
	release   chan struct{}
}

func (s *stubTiles) StateBatchDense(uint32, uint32) (clicks.DenseBatch, error) {
	s.reads.Add(1)
	if s.release != nil {
		s.entered <- struct{}{}
		<-s.release
	}
	return s.now, nil
}

func (s *stubTiles) Stage() clicks.Stage {
	if s.playing == (clicks.Stage{}) {
		return clicks.WorldStage
	}
	return s.playing
}

func (s *stubTiles) Kept(context.Context, clicks.Stage) (clicks.DenseBatch, error) {
	s.reads.Add(1)
	return s.kept, nil
}

func mapOf(owners ...string) clicks.DenseBatch {
	batch := clicks.DenseBatch{Start: 1, Codes: []string{""}}
	codes := map[string]uint16{"": 0}
	for _, owner := range owners {
		code, known := codes[owner]
		if !known {
			code = uint16(len(batch.Codes))
			codes[owner] = code
			batch.Codes = append(batch.Codes, owner)
		}
		batch.Tiles = binary.LittleEndian.AppendUint16(batch.Tiles, code)
	}
	return batch
}

func (s *testSuite) SetupSuite() {
	s.db = server.CreateDatabase(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_ledger_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(eras.Reseed(s.T().Context(), s.db))
	s.clock = cptime.NewFixedClock(now)
	s.tiles = &stubTiles{}
}

func (s *testSuite) query() *recent_history_query.PostgresQuery {
	forgettings := forgetting.NewPostgresQuery(s.db)
	world := world_replay.NewPostgresReplay(s.db, postgres_era_store.New(s.db), noCarries{})
	return recent_history_query.NewPostgresQuery(s.db, s.tiles, postgres_era_store.New(s.db), postgres_reset_store.New(s.db),
		forgettings, world_replay.NewErasures(world, forgettings, s.clock), s.clock)
}

type noCarries struct{}

func (noCarries) Carry(eras.Board, eras.Board) (ledger.Carry, error) { return noCarry{}, nil }

type noCarry struct{}

func (noCarry) Each(func(child, parent uint32)) {}

func (s *testSuite) moveHead(head ledger.Position) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE ledger_head SET head = $1`, int64(head)) //nolint:gosec // a few events.
	s.Require().NoError(err)
}

func (s *testSuite) record(forgotten map[ledger.Caller]ledger.Position, events ...ledger.Event) {
	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		s.Require().NoError(err)
		entry.Stage = ledger.StageOf(event)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i + 1), Entry: entry}) //nolint:gosec // a few events.
	}
	s.Require().NoError(s.store.Save(s.T().Context(), inmemory_ledger_storage.Changes{
		Entries: slices.Values(stored), Marks: inmemory_ledger_storage.Marks{Forgotten: forgotten},
	}))
}

func frames(from time.Time, changed map[time.Time][]*planetv1.TileOwner) []*planetv1.MapFrame {
	all := []*planetv1.MapFrame{{At: timestamppb.New(from), Whole: wholeOf(changed[from])}}
	for at := from.Add(30 * time.Minute); !at.After(until); at = at.Add(30 * time.Minute) {
		owners := changed[at]
		if owners == nil {
			owners = []*planetv1.TileOwner{}
		}
		all = append(all, &planetv1.MapFrame{At: timestamppb.New(at), Owners: owners})
	}
	return all
}

func wholeOf(owners []*planetv1.TileOwner) *planetv1.WholeMap {
	held := make(map[uint32]ledger.Owner, len(owners))
	for _, owner := range owners {
		held[owner.GetTile()] = ledger.Owner{Country: owner.GetCountryId()}
	}
	return planetmessage.Whole(held)
}

func take(account string, tile uint32, previous, country string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: "203.0.113.7", Account: account, Country: country, Previous: previous, At: at}
}

func (s *testSuite) TestTheLastDayIsRewoundFromTheMapNow() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("de", "fr", "", "jp", "cz")
	s.record(map[ledger.Caller]ledger.Position{{Account: eve}: 6},
		take(ada, 1, "fr", "de", since.Add(10*time.Minute)),
		ledger.Spreading{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "de", At: since.Add(10 * time.Minute), Impacts: []clicks.Impact{
			{Tile: 2, Owner: "it", Outcome: clicks.Taken},
		}},
		take(eve, 4, "jp", "ru", since.Add(time.Hour)),
		take(bob, 2, "de", "fr", since.Add(2*time.Hour+5*time.Minute)),
		ledger.Bombing{Scope: "203.0.113.8", Account: bob, At: until.Add(-10 * time.Minute), Blast: clicks.Blast{
			Tile: 3, CountryID: "fr", Cleared: []uint32{3}, Owners: []string{"jp"},
		}},
		take(ada, 5, "pl", "cz", until.Add(5*time.Minute)),
	)

	history, next, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.Equal(until.Add(30*time.Minute), next)
	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(since, map[time.Time][]*planetv1.TileOwner{
			since: {
				{Tile: 1, CountryId: "fr"},
				{Tile: 2, CountryId: "it"},
				{Tile: 3, CountryId: "jp"},
				{Tile: 4, CountryId: "jp"},
				{Tile: 5, CountryId: "pl"},
			},
			since.Add(30 * time.Minute): {
				{Tile: 1, CountryId: "de", SecondsIntoStep: 600},
				{Tile: 2, CountryId: "de", SecondsIntoStep: 600},
			},
			since.Add(150 * time.Minute): {{Tile: 2, CountryId: "fr", SecondsIntoStep: 300}},
			until:                        {{Tile: 3, SecondsIntoStep: 1200}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}, {Tile: 2, Hits: 2}, {Tile: 3, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "a reverted take is in none of it, and one after the end only rewinds: %v", history)
}

func (s *testSuite) TestARevertedDrawingPaintedOverOpensOnWhatWasBeneathIt() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("ie", "fr")
	s.record(map[ledger.Caller]ledger.Position{{Account: eve}: 3},
		take(eve, 1, "ie", "cn", since.Add(-time.Hour)),
		take(eve, 2, "ie", "cn", since.Add(-time.Hour)),
		take(ada, 1, "cn", "ie", since.Add(10*time.Minute)),
		take(bob, 2, "cn", "fr", since.Add(10*time.Minute)),
	)

	history, _, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(since, map[time.Time][]*planetv1.TileOwner{
			since:                       {{Tile: 1, CountryId: "ie"}, {Tile: 2, CountryId: "ie"}},
			since.Add(30 * time.Minute): {{Tile: 2, CountryId: "fr", SecondsIntoStep: 600}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 2, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "the drawing before the day never shows, and Ireland painting over it changed nothing: %v", history)
}

func (s *testSuite) TestADrawingForgottenBeforeTheHeadAndPaintedOverInTheDayOpensOnWhatWasBeneathIt() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("ie", "fr")
	s.record(map[ledger.Caller]ledger.Position{{Account: eve}: 3},
		take(eve, 1, "ie", "cn", since.Add(-80*time.Hour)),
		take(eve, 2, "ie", "cn", since.Add(-80*time.Hour)),
		take(ada, 1, "cn", "ie", since.Add(10*time.Minute)),
		take(bob, 2, "cn", "fr", since.Add(10*time.Minute)),
	)
	s.moveHead(3)

	history, _, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(since, map[time.Time][]*planetv1.TileOwner{
			since:                       {{Tile: 1, CountryId: "ie"}, {Tile: 2, CountryId: "ie"}},
			since.Add(30 * time.Minute): {{Tile: 2, CountryId: "fr", SecondsIntoStep: 600}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 2, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "the drawing behind the ledger's head never shows either: %v", history)
}

func (s *testSuite) TestARevertReadsWhatItForgotBeforeTheHeadAgain() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("fr")
	s.record(map[ledger.Caller]ledger.Position{},
		take(eve, 1, "ie", "cn", since.Add(-80*time.Hour)),
		take(bob, 1, "cn", "fr", since.Add(10*time.Minute)),
	)
	s.moveHead(2)
	query := s.query()
	drawn, _, err := query.RecentHistory(ctx)
	s.Require().NoError(err)
	s.True(proto.Equal(wholeOf([]*planetv1.TileOwner{{Tile: 1, CountryId: "cn"}}), drawn.GetFrames()[0].GetWhole()), "%v", drawn)

	s.Require().NoError(s.store.Save(ctx, inmemory_ledger_storage.Changes{
		From:    3,
		Entries: slices.Values([]inmemory_ledger_storage.Stored{}),
		Marks:   inmemory_ledger_storage.Marks{Head: 2, Forgotten: map[ledger.Caller]ledger.Position{{Account: eve}: 3}},
	}))
	erased, _, err := query.RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(since, map[time.Time][]*planetv1.TileOwner{
			since:                       {{Tile: 1, CountryId: "ie"}},
			since.Add(30 * time.Minute): {{Tile: 1, CountryId: "fr", SecondsIntoStep: 600}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}},
		Eras: worldOnEarth1,
	}, erased), "%v", erased)
}

func (s *testSuite) TestARevertBuildsTheStepsHistoryAgain() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("cn")
	s.record(map[ledger.Caller]ledger.Position{}, take(eve, 1, "ie", "cn", since.Add(50*time.Minute)))
	query := s.query()
	drawn, _, err := query.RecentHistory(ctx)
	s.Require().NoError(err)
	s.Len(drawn.GetFrames()[2].GetOwners(), 1)

	s.tiles.now = mapOf("ie")
	s.Require().NoError(s.store.Save(ctx, inmemory_ledger_storage.Changes{
		From:    2,
		Entries: slices.Values([]inmemory_ledger_storage.Stored{}),
		Marks:   inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{{Account: eve}: 2}},
	}))
	erased, _, err := query.RecentHistory(ctx)
	s.Require().NoError(err)

	s.Equal(int32(2), s.tiles.reads.Load(), "built again within the step")
	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(since, map[time.Time][]*planetv1.TileOwner{since: {{Tile: 1, CountryId: "ie"}}}),
		Eras:   worldOnEarth1,
	}, erased), "%v", erased)
}

func (s *testSuite) TestTheHistoryIsBuiltOncePerStepForEveryCaller() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("fr")
	s.tiles.entered, s.tiles.release = make(chan struct{}, 1), make(chan struct{})
	query := s.query()

	var (
		asked   sync.WaitGroup
		waiting atomic.Int32
		answers = make([]*planetv1.MapHistory, 5)
	)
	for i := range answers {
		asked.Go(func() {
			waiting.Add(1)
			history, _, err := query.RecentHistory(ctx)
			s.NoError(err)
			answers[i] = history
		})
	}
	<-s.tiles.entered
	s.Eventually(func() bool { return waiting.Load() == int32(len(answers)) }, time.Second, time.Millisecond)
	close(s.tiles.release)
	asked.Wait()

	s.Equal(int32(1), s.tiles.reads.Load())
	for _, answer := range answers {
		s.Same(answers[0], answer)
	}
	kept, _, err := query.RecentHistory(ctx)
	s.Require().NoError(err)
	s.Same(answers[0], kept)
	s.Equal(int32(1), s.tiles.reads.Load(), "kept until the next step")

	s.tiles.entered, s.tiles.release = make(chan struct{}, 1), nil
	s.clock.Advance(13 * time.Minute)
	next, at, err := query.RecentHistory(ctx)
	s.Require().NoError(err)
	s.Equal(int32(2), s.tiles.reads.Load(), "the next step builds its own")
	s.Equal(timestamppb.New(until.Add(30*time.Minute)), next.GetFrames()[len(next.GetFrames())-1].GetAt())
	s.Equal(until.Add(time.Hour), at)
}

func (s *testSuite) TestAResetInTheDayStartsTheHistoryAtTheFirstStepAfterIt() {
	ctx := s.T().Context()
	s.tiles.now = mapOf("de")
	s.record(map[ledger.Caller]ledger.Position{},
		take(ada, 1, "", "fr", until.Add(-80*time.Minute)),
		take(ada, 1, "fr", "de", until.Add(-20*time.Minute)),
	)
	_, err := s.db.On(ctx).ExecContext(ctx, `INSERT INTO map_resets (id, at) VALUES ('season-2', $1)`, until.Add(-70*time.Minute))
	s.Require().NoError(err)

	history, _, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(until.Add(-time.Hour), map[time.Time][]*planetv1.TileOwner{
			until.Add(-time.Hour): {{Tile: 1, CountryId: "fr"}},
			until:                 {{Tile: 1, CountryId: "de", SecondsIntoStep: 600}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "%v", history)
}

func (s *testSuite) TestAResetAfterTheDaysEndLeavesNothingOfIt() {
	ctx := s.T().Context()
	_, err := s.db.On(ctx).ExecContext(ctx, `INSERT INTO map_resets (id, at) VALUES ('season-2', $1)`, until.Add(time.Minute))
	s.Require().NoError(err)

	history, next, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{}, history), "%v", history)
	s.Equal(until.Add(30*time.Minute), next)
	s.Equal(int32(0), s.tiles.reads.Load())
}

func (s *testSuite) TestWhileAnArenaPlaysTheWorldsMapKeptIsRewoundAndTheArenasTakesLeftOut() {
	ctx := s.T().Context()
	s.tiles.now, s.tiles.kept = mapOf("br"), mapOf("de")
	s.tiles.playing = clicks.MapID("battle").On(clicks.TestBoard)
	s.record(map[ledger.Caller]ledger.Position{},
		take(ada, 1, "fr", "de", until.Add(-time.Hour)),
		ledger.Staged{On: s.tiles.playing, Event: take(bob, 1, "", "br", until.Add(-10*time.Minute))},
	)

	history, _, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(since, map[time.Time][]*planetv1.TileOwner{
			since:                        {{Tile: 1, CountryId: "fr"}},
			until.Add(-30 * time.Minute): {{Tile: 1, CountryId: "de"}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}},
		Eras: worldOnEarth1,
	}, history), "%v", history)
}

func (s *testSuite) TestASwitchOfTheWorldsBoardInTheDayStartsTheHistoryAtTheFirstStepAfterIt() {
	ctx := s.T().Context()
	switched := until.Add(-70 * time.Minute)
	earth2 := clicks.World.On(eras.Earth2.ID)
	s.Require().NoError(postgres_era_store.New(s.db).Start(ctx, earth2, switched))
	s.tiles.now = mapOf("de", "", "it")
	s.record(map[ledger.Caller]ledger.Position{},
		take(ada, 1, "", "jp", until.Add(-2*time.Hour)),
		ledger.Staged{On: earth2, Event: take(ada, 1, "fr", "pl", until.Add(-65*time.Minute))},
		ledger.Staged{On: earth2, Event: take(ada, 1, "pl", "de", until.Add(-20*time.Minute))},
		ledger.Staged{On: earth2, Event: take(bob, 3, "", "it", until.Add(-50*time.Minute))},
	)

	history, _, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{
		Frames: frames(until.Add(-time.Hour), map[time.Time][]*planetv1.TileOwner{
			until.Add(-time.Hour):        {{Tile: 1, CountryId: "pl"}},
			until.Add(-30 * time.Minute): {{Tile: 3, CountryId: "it", SecondsIntoStep: 600}},
			until:                        {{Tile: 1, CountryId: "de", SecondsIntoStep: 600}},
		}),
		Hits: []*planetv1.TileHits{{Tile: 1, Hits: 1}, {Tile: 3, Hits: 1}},
		Eras: []*planetv1.BoardEra{{StartedAt: timestamppb.New(switched), Board: "earth-2"}},
	}, history), "as after a reset: the first frame is earth-2's map rewound from now, the take on earth-1 before "+
		"is on another board, and so is every frame before the switch: %v", history)
}

func (s *testSuite) TestASwitchAfterTheDaysEndLeavesNothingOfIt() {
	ctx := s.T().Context()
	s.Require().NoError(postgres_era_store.New(s.db).Start(ctx, clicks.World.On(eras.Earth2.ID), until.Add(time.Minute)))

	history, _, err := s.query().RecentHistory(ctx)
	s.Require().NoError(err)

	s.True(proto.Equal(&planetv1.MapHistory{}, history), "the day was on earth-1, which the map no longer plays: %v", history)
	s.Equal(int32(0), s.tiles.reads.Load())
}

// The world's one era, as the migration that made eras keeps it.
var worldOnEarth1 = []*planetv1.BoardEra{{StartedAt: timestamppb.New(eras.Epoch), Board: "earth-1"}}
