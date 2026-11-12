package player_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeAccounts struct {
	created map[players.AccountID]time.Time
	err     error
}

func (f *fakeAccounts) CreatedAt(_ context.Context, account players.AccountID) (time.Time, error) {
	return f.created[account], f.err
}

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	players  *postgres_player_store.Store
	accounts *fakeAccounts
}

var (
	ada       = players.AccountID{15: 1}
	monday    = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	createdAt = time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.accounts = &fakeAccounts{created: map[players.AccountID]time.Time{ada: createdAt}}
	s.named(ada, "Ada_L")
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, monday)))
}

func (s *testSuite) query() *player_query.PostgresQuery {
	return player_query.NewPostgresQuery(s.db, s.accounts)
}

func (s *testSuite) player(account players.AccountID) *playerv1.Player {
	answer, err := s.query().Player(s.T().Context(), account)
	s.Require().NoError(err)
	return answer.GetPlayer()
}

func (s *testSuite) TestAPlayerIsFoundByItsAccount() {
	s.True(proto.Equal(&playerv1.Player{AccountId: ada.String(), Name: "Ada_L", CreatedAtUnixMs: createdAt.UnixMilli()}, s.player(ada)))
}

func (s *testSuite) TestAnAccountKeepsItsCardWhenAnotherTakesTheNameItLeft() {
	bob := players.AccountID{15: 2}
	s.named(ada, "Ada_Lovelace")
	s.named(bob, "Ada_L")

	s.Equal("Ada_Lovelace", s.player(ada).GetName())
	s.Equal("Ada_L", s.player(bob).GetName())
}

func (s *testSuite) TestAGuestIsItsAccountAndItsNameAlone() {
	guest := players.AccountID{15: 3}
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))

	answer, err := s.query().Player(s.T().Context(), guest)

	s.Require().NoError(err)
	s.True(proto.Equal(&playerv1.Player{AccountId: guest.String(), Name: "guest_a1b2c3", Guest: true}, answer.GetPlayer()),
		"%v", answer.GetPlayer())
}

func (s *testSuite) TestAnAccountTheGameNeverNamedIsNoPlayer() {
	_, err := s.query().Player(s.T().Context(), players.AccountID{15: 5})

	s.ErrorIs(err, player_query.ErrNoPlayer)
}

func (s *testSuite) TestAnAccountAskedToBeDeletedIsNoPlayerUntilItComesBack() {
	store := postgres_departure_store.New(s.db)
	decide := func(decision departures.Decision) {
		s.Require().NoError(store.Change(s.T().Context(), ada,
			func(d departures.Departure) departures.Departure { return d.Decided(decision) }))
	}
	decide(departures.Leaves(monday))

	_, err := s.query().Player(s.T().Context(), ada)
	s.Require().ErrorIs(err, player_query.ErrNoPlayer, "its card reads as a deleted player's")

	decide(departures.Returns(monday.Add(time.Hour)))

	s.Equal("Ada_L", s.player(ada).GetName())
}

func (s *testSuite) TestAnAdminIsSaidToBeOne() {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	s.True(s.player(ada).GetAdmin())
}

func (s *testSuite) TestThePlayerCarriesItsColor() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))

	s.Equal(playerv1.NameColor_NAME_COLOR_TEAL, s.player(ada).GetColor())
}

func (s *testSuite) TestAnAccountAuthNoLongerKnowsHasNoCreationDate() {
	bob := players.AccountID{15: 2}
	s.named(bob, "Bob")

	s.Zero(s.player(bob).GetCreatedAtUnixMs())
}

func (s *testSuite) TestAColorTheProtoDoesNotNameIsAnError() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, 99))

	_, err := s.query().Player(s.T().Context(), ada)

	s.ErrorIs(err, playerread.ErrUnknownColor)
}

func (s *testSuite) TestAStoreFailureIsNotNoProfile() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Player(ctx, ada)

	s.Require().Error(err)
	s.NotErrorIs(err, player_query.ErrNoPlayer)
}

func (s *testSuite) TestAnAuthFailureIsAnError() {
	s.accounts.err = errors.New("auth is down")

	_, err := s.query().Player(s.T().Context(), ada)

	s.Require().ErrorIs(err, s.accounts.err)
	s.NotErrorIs(err, player_query.ErrNoPlayer)
}
