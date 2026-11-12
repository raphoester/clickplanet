package holder_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_by_name_handler/holder_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
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
	ada    = players.AccountID{15: 1}
	bob    = players.AccountID{15: 2}
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.named(ada, "Ada Lovelace")
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, monday)))
}

func (s *testSuite) holder(name string) (*playerv1.Player, error) {
	answer, err := holder_query.NewPostgresQuery(s.db).Holder(s.T().Context(), players.FoldedOf(name))
	return answer.GetPlayer(), err
}

func (s *testSuite) TestTheHolderOfANameIsFoundInAnyCase() {
	for _, asked := range []string{"Ada Lovelace", "ada lovelace", "ADA LOVELACE", "Ａｄａ Ｌｏｖｅｌａｃｅ"} {
		player, err := s.holder(asked)

		s.Require().NoError(err, asked)
		s.True(proto.Equal(&playerv1.Player{AccountId: ada.String(), Name: "Ada Lovelace"}, player), "%q: %v", asked, player)
	}
}

func (s *testSuite) TestOnlyTheWholeNameFindsItsHolder() {
	for _, asked := range []string{"Ada", "Ada Lovelace2", "Ada-Lovelace", " Ada Lovelace", "Ada  Lovelace"} {
		_, err := s.holder(asked)

		s.ErrorIs(err, holder_query.ErrNoHolder, asked)
	}
}

func (s *testSuite) TestANameLeftByARenameIsTheNewHolders() {
	s.named(ada, "Countess")
	s.named(bob, "Ada Lovelace")

	player, err := s.holder("Ada Lovelace")

	s.Require().NoError(err)
	s.Equal(bob.String(), player.GetAccountId())
}

func (s *testSuite) TestAGuestCodeHoldsNoName() {
	guest := players.AccountID{15: 3}
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))

	_, err := s.holder("guest_a1b2c3")

	s.ErrorIs(err, holder_query.ErrNoHolder)
}

func (s *testSuite) TestAnAccountAskedToBeDeletedHoldsNoNameUntilItComesBack() {
	store := postgres_departure_store.New(s.db)
	decide := func(decision departures.Decision) {
		s.Require().NoError(store.Change(s.T().Context(), ada,
			func(d departures.Departure) departures.Departure { return d.Decided(decision) }))
	}
	decide(departures.Leaves(monday))

	_, err := s.holder("Ada Lovelace")
	s.Require().ErrorIs(err, holder_query.ErrNoHolder)

	decide(departures.Returns(monday.Add(time.Hour)))

	player, err := s.holder("Ada Lovelace")
	s.Require().NoError(err)
	s.Equal(ada.String(), player.GetAccountId())
}

func (s *testSuite) TestTheHolderCarriesItsColorAndCrown() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	player, err := s.holder("Ada Lovelace")

	s.Require().NoError(err)
	s.Equal(playerv1.NameColor_NAME_COLOR_TEAL, player.GetColor())
	s.True(player.GetAdmin())
}

func (s *testSuite) TestAColorTheProtoDoesNotNameIsAnError() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, 99))

	_, err := s.holder("Ada Lovelace")

	s.ErrorIs(err, playerread.ErrUnknownColor)
}

func (s *testSuite) TestAStoreFailureIsNotNoHolder() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := holder_query.NewPostgresQuery(s.db).Holder(ctx, players.FoldedOf("Ada Lovelace"))

	s.Require().Error(err)
	s.NotErrorIs(err, holder_query.ErrNoHolder)
}
