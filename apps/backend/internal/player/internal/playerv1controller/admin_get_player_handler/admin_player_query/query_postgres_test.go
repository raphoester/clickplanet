package admin_player_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler/admin_player_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeAccounts struct {
	kept map[players.AccountID]players.Account
	err  error
}

func (f *fakeAccounts) Account(_ context.Context, account players.AccountID) (players.Account, error) {
	return f.kept[account], f.err
}

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	players  *postgres_player_store.Store
	accounts *fakeAccounts
}

var (
	ada       = players.AccountID{15: 1}
	guest     = players.AccountID{15: 2}
	monday    = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	createdAt = time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.accounts = &fakeAccounts{kept: map[players.AccountID]players.Account{
		ada:   players.AccountOf(true, createdAt),
		guest: players.AccountOf(false, createdAt),
	}}
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(ada, "Ada_L", monday)))
}

func (s *testSuite) query() *admin_player_query.PostgresQuery {
	return admin_player_query.NewPostgresQuery(s.db, s.accounts)
}

func (s *testSuite) player(account players.AccountID) *playerv1.AdminGetPlayerResponse {
	answer, err := s.query().Player(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestAUsernameIsReadWithWhoItIs() {
	ctx := s.T().Context()
	s.Require().NoError(s.players.SaveColor(ctx, ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))
	_, err := s.db.On(ctx).ExecContext(ctx, `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	s.True(proto.Equal(&playerv1.AdminGetPlayerResponse{
		Name:            "Ada_L",
		Linked:          true,
		Admin:           true,
		Color:           playerv1.NameColor_NAME_COLOR_TEAL,
		CreatedAtUnixMs: createdAt.UnixMilli(),
	}, s.player(ada)), "%v", s.player(ada))
}

func (s *testSuite) TestAGuestIsReadUnderItsCode() {
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))

	player := s.player(guest)

	s.Equal("guest_a1b2c3", player.GetName())
	s.True(player.GetGuest())
	s.False(player.GetLinked())
	s.Equal(createdAt.UnixMilli(), player.GetCreatedAtUnixMs())
}

func (s *testSuite) TestAnAccountNeverShownHasNoNameButIsOneWhileAuthKnowsIt() {
	player := s.player(guest)

	s.Empty(player.GetName(), "it never announced and never posted, so it was never given a code")
	s.True(player.GetGuest())
	s.Equal(createdAt.UnixMilli(), player.GetCreatedAtUnixMs())
}

func (s *testSuite) TestAnAccountTheGameAndAuthKnowNothingAboutIsNoPlayer() {
	_, err := s.query().Player(s.T().Context(), players.AccountID{15: 9})

	s.ErrorIs(err, admin_player_query.ErrNoPlayer)
}

func (s *testSuite) TestAnAccountAuthNoLongerKnowsHasNoCreationDate() {
	delete(s.accounts.kept, ada)

	player := s.player(ada)

	s.Zero(player.GetCreatedAtUnixMs())
	s.False(player.GetLinked())
}

func (s *testSuite) TestAFailureToAskAuthIsAnError() {
	s.accounts.err = errors.New("down")

	_, err := s.query().Player(s.T().Context(), ada)

	s.Require().ErrorIs(err, s.accounts.err)
	s.NotErrorIs(err, admin_player_query.ErrNoPlayer)
}

func (s *testSuite) TestAStoreFailureIsNotNoPlayer() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Player(ctx, ada)

	s.Require().Error(err)
	s.NotErrorIs(err, admin_player_query.ErrNoPlayer)
}
