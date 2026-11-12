package graffiti_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	vandalsquadv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/postgres_graffiti_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler/graffiti_query"
)

const (
	ana  = "01900000-0000-7000-8000-000000000001"
	link = "https://clickplanet.lol/play?lat=1.00&lon=2.00&zoom=20.0"
)

var at = time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "vandalsquad", migrations.FS)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func graffitiOf(t *testing.T, id byte, account cpsession.AccountID, flagged time.Time) graffiti.Graffiti {
	t.Helper()
	made, err := graffiti.Snapshot{
		ID: graffiti.ID{15: id}, Account: account, Flag: "ro", Kind: graffiti.Swastika, Tiles: []drawings.TileID{2, 1},
		FlaggedAt: flagged, SeenAt: flagged.Add(time.Minute), Link: link,
	}.Restore()
	require.NoError(t, err)
	return made
}

func (s *testSuite) TestTheGraffitiAreListedNewestFirstAndAForgottenPaintersAsNobodys() {
	t, db := s.T(), s.db
	painter := cpsession.AccountID(uuid.MustParse(ana))
	require.NoError(t, postgres_graffiti_store.New(db).Save(t.Context(),
		graffitiOf(t, 1, painter, at),
		graffitiOf(t, 2, painter, at.Add(time.Hour)).Anonymized(),
	))

	listed, err := graffiti_query.NewPostgresQuery(db).Graffiti(t.Context())

	require.NoError(t, err)
	listing := func(account string, flagged time.Time) *vandalsquadv1.Graffiti {
		return &vandalsquadv1.Graffiti{
			AccountId: account, CountryId: "ro", Kind: vandalsquadv1.GraffitiKind_GRAFFITI_KIND_SWASTIKA,
			TileIds: []uint32{2, 1}, FlaggedAt: timestamppb.New(flagged), SeenAt: timestamppb.New(flagged.Add(time.Minute)),
			Link: link,
		}
	}
	want := &vandalsquadv1.ListGraffitiResponse{Graffiti: []*vandalsquadv1.Graffiti{
		listing("", at.Add(time.Hour)),
		listing(ana, at),
	}}
	assert.True(t, proto.Equal(want, listed), "got %v", listed)
}

func (s *testSuite) TestTheListStopsAtTheNewestFiveHundred() {
	t, db := s.T(), s.db
	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO graffiti (id, account, flag, kind, tiles, flagged_at, seen_at, link)
		SELECT gen_random_uuid(), NULL, 'ro', 'swastika', '{1}', $1::timestamptz + n * interval '1 minute',
			$1::timestamptz + n * interval '1 minute', 'https://clickplanet.lol/play'
		FROM generate_series(1, 501) AS n
	`, at)
	require.NoError(t, err)

	listed, err := graffiti_query.NewPostgresQuery(db).Graffiti(t.Context())

	require.NoError(t, err)
	require.Len(t, listed.GetGraffiti(), graffiti_query.Listed)
	assert.Equal(t, at.Add(501*time.Minute), listed.GetGraffiti()[0].GetFlaggedAt().AsTime())
}
