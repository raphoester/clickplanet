package profile_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_profile_handler/profile_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	players *postgres_player_store.Store
}

var (
	ada = players.AccountID{15: 1}
	at  = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) profile(account players.AccountID) *playerv1.GetProfileResponse {
	answer, err := profile_query.NewPostgresQuery(s.db).Profile(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestAnAccountThatNeverChoseANameHasAnEmptyOne() {
	answer := s.profile(ada)

	s.Equal(ada.String(), answer.GetProfile().GetAccountId())
	s.Empty(answer.GetProfile().GetName())
	s.Equal(playerv1.NameColor_NAME_COLOR_UNSPECIFIED, answer.GetColor())
}

func (s *testSuite) TestTheChosenNameAndColorAreAnswered() {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(ada, "Ada_L", at)))
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))

	answer := s.profile(ada)

	s.Equal("Ada_L", answer.GetProfile().GetName())
	s.Equal(playerv1.NameColor_NAME_COLOR_TEAL, answer.GetColor())
}

func (s *testSuite) TestAColorTheProtoDoesNotNameIsAnError() {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(ada, "Ada_L", at)))
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, 99))

	_, err := profile_query.NewPostgresQuery(s.db).Profile(s.T().Context(), ada)

	s.ErrorIs(err, playerread.ErrUnknownColor)
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := profile_query.NewPostgresQuery(s.db).Profile(ctx, ada)

	s.Error(err, "not an empty profile")
}
