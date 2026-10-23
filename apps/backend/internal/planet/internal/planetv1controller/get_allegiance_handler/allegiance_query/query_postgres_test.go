package allegiance_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_handler/allegiance_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_allegiance_store.Store
	query *allegiance_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_allegiance_store.New(s.db)
	s.query = allegiance_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

const (
	alice allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

var changed = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func (s *testSuite) keep(account allegiances.Account, kept allegiances.Snapshot) {
	s.Require().NoError(s.store.Change(s.T().Context(), account, func(allegiances.Allegiance) (allegiances.Allegiance, error) {
		return kept.Restore(), nil
	}))
}

func (s *testSuite) allegiance(account allegiances.Account) *planetv1.GetAllegianceResponse {
	answer, err := s.query.Allegiance(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestAnAllegianceSaysWhenItMayChange() {
	s.keep(alice, allegiances.Snapshot{Flag: "de", ChangedAt: changed})

	s.True(proto.Equal(&planetv1.GetAllegianceResponse{Allegiance: &planetv1.Allegiance{
		CountryId: "de", ChangeableAtUnixMs: changed.Add(time.Hour).UnixMilli(),
	}}, s.allegiance(alice)))
}

func (s *testSuite) TestAFirstFlagMayChangeNow() {
	s.keep(alice, allegiances.Snapshot{Flag: "fr"})

	s.True(proto.Equal(&planetv1.GetAllegianceResponse{Allegiance: &planetv1.Allegiance{CountryId: "fr"}},
		s.allegiance(alice)))
}

func (s *testSuite) TestNoFlagYetAndNoAccountAnswerNoAllegiance() {
	s.keep(alice, allegiances.Snapshot{Flag: "fr"})

	s.Nil(s.allegiance(bob).GetAllegiance())
	s.Nil(s.allegiance(allegiances.NoAccount).GetAllegiance())
}
