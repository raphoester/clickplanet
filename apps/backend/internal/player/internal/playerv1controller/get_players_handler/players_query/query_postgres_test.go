package players_query_test

import (
	"context"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/introduce_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_players_handler/players_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
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
	ada   = players.AccountID{15: 1}
	bob   = players.AccountID{15: 2}
	guest = players.AccountID{15: 3}
	today = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.named(ada, "Ada")
	s.named(bob, "Bob")
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, today)))
}

func (s *testSuite) query() *players_query.PostgresQuery {
	return players_query.NewPostgresQuery(s.db)
}

func (s *testSuite) answered(accounts ...players.AccountID) map[string]*playerv1.Player {
	answer, err := s.query().Players(s.T().Context(), accounts)
	s.Require().NoError(err)

	by := make(map[string]*playerv1.Player, len(answer.GetPlayers()))
	for _, player := range answer.GetPlayers() {
		by[player.GetAccountId()] = player
	}
	return by
}

func names(found map[string]*playerv1.Player) map[string]string {
	by := make(map[string]string, len(found))
	for account, player := range found {
		by[account] = player.GetName()
	}
	return by
}

func (s *testSuite) TestAPlayerIsNamedByItsUsernameAndAGuestByItsCode() {
	s.Equal(map[string]string{ada.String(): "Ada", guest.String(): "guest_a1b2c3"}, names(s.answered(ada, guest)))
}

func (s *testSuite) TestAUsernameWinsOverTheGuestCodeItNoLongerGoesBy() {
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), ada, "91aa3d"))

	s.Equal(map[string]string{ada.String(): "Ada"}, names(s.answered(ada)))
}

func (s *testSuite) TestARenamedPlayerIsNamedByItsNewNameAndNotByWhoTookItsOldOne() {
	s.named(ada, "Ada_Lovelace")
	s.named(bob, "Ada")

	s.Equal(map[string]string{ada.String(): "Ada_Lovelace", bob.String(): "Ada"}, names(s.answered(ada, bob)))
}

func (s *testSuite) TestAGuestIsToldApartFromAUsername() {
	found := s.answered(ada, guest)

	s.False(found[ada.String()].GetGuest())
	s.True(found[guest.String()].GetGuest())
}

func (s *testSuite) TestAnAdminCarriesItsMark() {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	found := s.answered(ada, bob)

	s.True(found[ada.String()].GetAdmin())
	s.False(found[bob.String()].GetAdmin())
}

func (s *testSuite) TestAnAccountNobodyCanNameIsLeftOut() {
	unnamed := players.AccountID{15: 4}

	s.Equal(map[string]string{ada.String(): "Ada"}, names(s.answered(ada, unnamed, players.AccountID{15: 5})),
		"an account with neither a profile nor a code is absent, and so is an unknown one")
}

func (s *testSuite) TestAskingGivesNobodyAGuestCode() {
	unnamed := players.AccountID{15: 4}

	s.answered(unnamed)

	_, err := s.players.GuestCode(s.T().Context(), unnamed)
	s.ErrorIs(err, players.ErrNoGuestCode, "a read path never writes")
}

func (s *testSuite) TestADeletedAccountCanNoLongerBeNamed() {
	s.Require().NoError(s.players.DeleteAccount(s.T().Context(), ada))

	s.Empty(s.answered(ada), "what a caller shows in its place is the caller's to decide")
}

func (s *testSuite) decide(account players.AccountID, decision departures.Decision) {
	s.Require().NoError(postgres_departure_store.New(s.db).Change(s.T().Context(), account,
		func(d departures.Departure) departures.Departure { return d.Decided(decision) }))
}

func (s *testSuite) TestAnAccountAskedToBeDeletedReadsAsDeletedUntilItComesBack() {
	s.decide(ada, departures.Leaves(today))

	s.Equal(map[string]string{bob.String(): "Bob"}, names(s.answered(ada, bob)), "the other players see it gone")

	s.decide(ada, departures.Returns(today.Add(time.Hour)))

	s.Equal(map[string]string{ada.String(): "Ada", bob.String(): "Bob"}, names(s.answered(ada, bob)), "back under its name")
}

func (s *testSuite) TestAskingAboutNobodyAnswersNobody() {
	s.Empty(s.answered())
}

func (s *testSuite) TestARepeatedAccountIsAnsweredOnceInTheOrderAsked() {
	answer, err := s.query().Players(s.T().Context(), []players.AccountID{guest, ada, guest, bob})
	s.Require().NoError(err)

	ids := make([]string, 0, len(answer.GetPlayers()))
	for _, player := range answer.GetPlayers() {
		ids = append(ids, player.GetAccountId())
	}
	s.Equal([]string{guest.String(), ada.String(), bob.String()}, ids)
}

func (s *testSuite) TestEachPlayerCarriesItsColorAndAGuestNone() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_PINK)))

	found := s.answered(ada, guest)

	s.Equal(playerv1.NameColor_NAME_COLOR_PINK, found[ada.String()].GetColor())
	s.Equal(playerv1.NameColor_NAME_COLOR_UNSPECIFIED, found[guest.String()].GetColor())
}

func (s *testSuite) TestAColorTheProtoDoesNotNameIsAnError() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, 99))

	_, err := s.query().Players(s.T().Context(), []players.AccountID{ada})

	s.ErrorIs(err, playerread.ErrUnknownColor)
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Players(ctx, []players.AccountID{ada})

	s.Error(err)
}

func (s *testSuite) TestEachPlayerIsWhatIntroducePlayerAnswersForIt() {
	cy, dan, eve := players.AccountID{15: 4}, players.AccountID{15: 5}, players.AccountID{15: 6}
	s.named(cy, "Cyd")
	s.named(dan, "Dan")
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), eve, "0d0e0f"))
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_PINK)))
	single := introduce_player_usecase.New(s.players, players.NewGuestCodes(s.players, &players.SequentialCodes{}))
	everyone := []players.AccountID{ada, bob, guest, cy, dan, eve}
	many := s.answered(everyone...)

	s.Require().Len(many, len(everyone))
	for _, account := range everyone {
		one, err := single.Execute(s.T().Context(), account)
		s.Require().NoError(err)

		s.True(proto.Equal(&playerv1.Player{
			AccountId: account.String(),
			Name:      one.Name(),
			Admin:     one.Admin(),
			Color:     playermessage.Color(one.Color()),
			Guest:     one.Guest(),
		}, many[account.String()]), "%s: IntroducePlayer and GetPlayers must name an account alike, got %v", one.Name(), many[account.String()])
	}
}
