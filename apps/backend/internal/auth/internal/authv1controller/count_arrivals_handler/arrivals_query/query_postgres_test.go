package arrivals_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/postgres_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/count_arrivals_handler/arrivals_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_arrival_store.Store
	next  byte
}

var labDay = time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_arrival_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) arrive(tag string, at time.Time) arrivals.Arrival {
	s.next++
	arrival, err := arrivals.NewArrival(arrivals.ArrivalID{15: s.next}, arrivals.AccountID{14: 1, 15: s.next},
		arrivals.Landing{Tag: tag}, at)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), arrival))
	return arrival
}

func (s *testSuite) counts(since, until time.Time) *authv1.CountArrivalsResponse {
	counted, err := arrivals_query.NewPostgresQuery(s.db).Counts(s.T().Context(), since, until)
	s.Require().NoError(err)
	return counted
}

func count(day string, source authv1.ArrivalSource, accounts uint32) *authv1.ArrivalCount {
	return &authv1.ArrivalCount{Day: day, Source: source, Accounts: accounts}
}

func (s *testSuite) TestEachUTCDayCountsItsAccountsBySourceTheOldestDayFirst() {
	s.arrive("t", labDay.Add(23*time.Hour+30*time.Minute))
	s.arrive("t", labDay.Add(24*time.Hour+30*time.Minute))
	s.arrive("y", labDay.Add(time.Hour))
	s.arrive("yt", labDay.Add(2*time.Hour))
	s.arrive("t", labDay.Add(3*time.Hour))
	s.arrive("", labDay.Add(4*time.Hour))
	s.arrive("r", labDay.Add(5*time.Hour))
	s.arrive("v", labDay.Add(6*time.Hour))

	counted := s.counts(time.Time{}, time.Time{})

	s.True(proto.Equal(&authv1.CountArrivalsResponse{Counts: []*authv1.ArrivalCount{
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_RECRUIT, 1),
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_RECRUIT_VIDEO, 1),
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_TIKTOK, 2),
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_UNKNOWN, 1),
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_YOUTUBE, 2),
		count("2026-11-21", authv1.ArrivalSource_ARRIVAL_SOURCE_TIKTOK, 1),
	}}, counted), counted)
}

func (s *testSuite) TestOnlyTheAccountsMadeInTheWindowCount() {
	s.arrive("t", labDay.Add(-time.Minute))
	s.arrive("t", labDay)
	s.arrive("t", labDay.Add(48*time.Hour-time.Minute))
	s.arrive("t", labDay.Add(48*time.Hour))

	counted := s.counts(labDay, labDay.Add(48*time.Hour))

	s.True(proto.Equal(&authv1.CountArrivalsResponse{Counts: []*authv1.ArrivalCount{
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_TIKTOK, 1),
		count("2026-11-21", authv1.ArrivalSource_ARRIVAL_SOURCE_TIKTOK, 1),
	}}, counted), counted)
}

func (s *testSuite) TestADeletedAccountsArrivalStillCounts() {
	gone := s.arrive("y", labDay.Add(time.Hour))
	s.Require().NoError(s.store.Save(s.T().Context(), gone.Anonymized()))

	counted := s.counts(time.Time{}, time.Time{})

	s.True(proto.Equal(&authv1.CountArrivalsResponse{Counts: []*authv1.ArrivalCount{
		count("2026-11-20", authv1.ArrivalSource_ARRIVAL_SOURCE_YOUTUBE, 1),
	}}, counted), counted)
}

func (s *testSuite) TestNoArrivalIsNoCount() {
	s.Empty(s.counts(time.Time{}, time.Time{}).GetCounts())
}

func (s *testSuite) TestASourceThisBuildDoesNotKnowRefusesTheRead() {
	ctx := s.T().Context()
	_, err := s.db.On(ctx).ExecContext(ctx, `ALTER TYPE arrival_source ADD VALUE IF NOT EXISTS 'instagram'`)
	s.Require().NoError(err)
	_, err = s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO arrivals (id, source, tag, referrer, arrived_at) VALUES (gen_random_uuid(), 'instagram', 'i', '', now())`)
	s.Require().NoError(err)

	_, err = arrivals_query.NewPostgresQuery(s.db).Counts(ctx, time.Time{}, time.Time{})

	s.Require().ErrorIs(err, arrivals_query.ErrUnknownSource)
}
