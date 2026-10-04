package log_query_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/read_log_handler/log_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	ledger *postgres_ledger_store.Store
	query  *log_query.PostgresQuery
}

const ada = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

var noon = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.ledger = postgres_ledger_store.New(s.db)
	s.query = log_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) save(from, head ledger.Position, forgotten map[ledger.Caller]ledger.Position, takes ...inmemory_ledger_storage.Stored) {
	s.Require().NoError(s.ledger.Save(s.T().Context(), inmemory_ledger_storage.Changes{
		From:  from,
		Takes: slices.Values(takes),
		Marks: inmemory_ledger_storage.Marks{Head: head, Forgotten: forgotten},
	}))
}

func stored(position ledger.Position, tile uint32, scope, account, country, previous string) inmemory_ledger_storage.Stored {
	return inmemory_ledger_storage.Stored{Position: position, Taking: ledger.Taking{
		Tile: tile, Scope: scope, Account: account, Country: country, Previous: previous, At: noon.Add(time.Duration(position) * time.Second),
	}}
}

func (s *testSuite) entries(from uint64, limit uint32) []*planetv1.LogEntry {
	answer, err := s.query.Entries(s.T().Context(), from, limit)
	s.Require().NoError(err)
	return answer.GetEntries()
}

func positions(entries []*planetv1.LogEntry) []uint64 {
	out := make([]uint64, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.GetPosition())
	}
	return out
}

func (s *testSuite) TestAnEmptyLedgerHasNoEntries() {
	s.Empty(s.entries(0, 10))
}

func (s *testSuite) TestATakeIsAnEntryWithAllItCarries() {
	s.save(0, 0, map[ledger.Caller]ledger.Position{},
		stored(0, 7, "1.2.3.4", ada, "fr", "de"),
		stored(1, 8, "1.2.3.4", "", "", "pl"),
	)

	entries := s.entries(0, 10)

	s.Require().Len(entries, 2)
	s.True(proto.Equal(&planetv1.LogEntry{Position: 0, Fact: &planetv1.LogEntry_Take{Take: &planetv1.Take{
		TileId: 7, AccountId: ada, Country: "fr", Previous: "de", TakenAt: timestamppb.New(noon),
	}}}, entries[0]), entries[0].String())
	s.True(proto.Equal(&planetv1.LogEntry{Position: 1, Fact: &planetv1.LogEntry_Take{Take: &planetv1.Take{
		TileId: 8, Previous: "pl", TakenAt: timestamppb.New(noon.Add(time.Second)),
	}}}, entries[1]), "a take with no account and a clear of native land: %s", entries[1].String())
}

func (s *testSuite) TestTheEntriesFromAPositionComeInPositionOrderAcrossFlushes() {
	s.save(0, 0, map[ledger.Caller]ledger.Position{}, stored(0, 1, "a", ada, "fr", ""), stored(1, 2, "a", ada, "fr", ""))
	s.save(5, 0, map[ledger.Caller]ledger.Position{}, stored(5, 3, "a", ada, "fr", ""), stored(6, 4, "a", ada, "fr", ""))

	s.Equal([]uint64{0, 1, 5, 6}, positions(s.entries(0, 10)))
	s.Equal([]uint64{1, 5, 6}, positions(s.entries(1, 10)), "the position asked is the first answered")
	s.Equal([]uint64{5, 6}, positions(s.entries(2, 10)), "a gap is skipped")
	s.Empty(s.entries(7, 10), "past the last take there is nothing yet")
}

func (s *testSuite) TestTheLimitCutsTheAnswerAndIsCapped() {
	all := make([]inmemory_ledger_storage.Stored, log_query.MaxEntries+5)
	for i := range all {
		all[i] = stored(ledger.Position(i), uint32(i+1), "a", ada, "fr", "")
	}
	s.save(0, 0, map[ledger.Caller]ledger.Position{}, all...)

	s.Equal([]uint64{0, 1, 2}, positions(s.entries(0, 3)))
	s.Len(s.entries(0, 0), log_query.MaxEntries, "no limit is the cap")
	s.Len(s.entries(0, log_query.MaxEntries+5), log_query.MaxEntries, "a limit past the cap is the cap")
}

func (s *testSuite) TestATakeLaterRevertedSaysSoAndKeepsSayingItPastTheHead() {
	s.save(0, 0, map[ledger.Caller]ledger.Position{},
		stored(0, 1, "bot", ada, "fr", ""),
		stored(1, 2, "player", "", "de", ""),
	)
	s.False(s.entries(0, 10)[0].GetTake().GetReverted())

	s.save(2, 0, map[ledger.Caller]ledger.Position{{Account: ada}: 2})
	s.save(2, 2, map[ledger.Caller]ledger.Position{})

	entries := s.entries(0, 10)
	s.True(entries[0].GetTake().GetReverted())
	s.False(entries[1].GetTake().GetReverted())
}

func (s *testSuite) TestADeletedAccountsTakesNameNoAccount() {
	s.save(0, 0, map[ledger.Caller]ledger.Position{}, stored(0, 1, "a", ada, "fr", ""))
	account, err := ledger.AccountIDOf(ada)
	s.Require().NoError(err)

	s.Require().NoError(s.ledger.AnonymizeTakes(s.T().Context(), account))

	s.Empty(s.entries(0, 10)[0].GetTake().GetAccountId())
}

func (s *testSuite) TestAPositionPastTheLargestBigintIsNothing() {
	s.save(0, 0, map[ledger.Caller]ledger.Position{}, stored(0, 1, "a", ada, "fr", ""))

	s.Empty(s.entries(1<<63+1, 10))
}
