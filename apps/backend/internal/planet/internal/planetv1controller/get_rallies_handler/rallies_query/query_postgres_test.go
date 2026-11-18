package rallies_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rallies_handler/rallies_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies/postgres_rally_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_rally_store.Store
	clock *cptime.FixedClock
	query *rallies_query.PostgresQuery
}

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_rally_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(eras.Reseed(s.T().Context(), s.db))
	s.clock = cptime.NewFixedClock(now)
	s.query = rallies_query.NewPostgresQuery(s.db, s.clock)
}

func rallyOf(n byte, flag string, calledAt time.Time) rallies.Snapshot {
	return rallies.Snapshot{
		ID:      rallies.RallyID(uuid.UUID{0, 0, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, n}),
		Account: cpsession.AccountID(uuid.UUID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0, 0, 0, 0, 0, n}),
		Flag:    flag, Centre: clicks.Vec3{X: 0.6, Y: float64(n) / 10, Z: 0.8}, Radius: 0.072, CalledAt: calledAt, EndsAt: calledAt.Add(3 * time.Minute),
	}
}

func (s *testSuite) keep(kept rallies.Snapshot) {
	s.Require().NoError(s.store.Change(s.T().Context(), kept.Account, func(rallies.Rally) (rallies.Rally, error) {
		return kept.Restore(), nil
	}))
}

func (s *testSuite) TestEveryRallyStillOnOfEveryFlagWithTheTimeItHasLeft() {
	s.keep(rallyOf(1, "fr", now.Add(-time.Minute)))
	s.keep(rallyOf(2, "de", now.Add(-2*time.Minute)))
	s.keep(rallyOf(3, "fr", now.Add(-3*time.Minute)))

	answer, err := s.query.Rallies(s.T().Context())
	s.Require().NoError(err)

	s.Require().Len(answer.GetRallies(), 2, "one that ends now is over")
	s.True(proto.Equal(&planetv1.Rally{
		RallyId: "00000000-0000-7000-8000-000000000002", AccountId: "01926c6e-7a4b-7c3d-8e9f-000000000002", CountryId: "de",
		Point: &planetv1.GlobePoint{X: 0.6, Y: 0.2, Z: 0.8}, Radius: 0.072, LeftMs: 60_000,
	}, answer.GetRallies()[0]), "the soonest to end first: %v", answer.GetRallies()[0])
	s.Equal(uint32(120_000), answer.GetRallies()[1].GetLeftMs())
	s.Equal("fr", answer.GetRallies()[1].GetCountryId())
}

func (s *testSuite) TestNoRallyOnIsAnEmptyAnswer() {
	answer, err := s.query.Rallies(s.T().Context())

	s.Require().NoError(err)
	s.Empty(answer.GetRallies())
}
