package account_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/find_account_handler/account_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler/authors_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	players *postgres_player_store.Store
	query   *account_query.PostgresQuery
}

var (
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
	named = players.AccountID{15: 3}
	today = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
	s.query = account_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(ada, "Ada_L", today)))
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), named, "d4e5f6"))
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(named, "Straße", today)))
}

func (s *testSuite) found(name string) string {
	answer, err := s.query.Account(s.T().Context(), name)
	s.Require().NoError(err, "%q", name)
	return answer.GetAccountId()
}

func (s *testSuite) TestAUsernameFindsItsAccountInAnyCase() {
	for _, name := range []string{"Ada_L", "ada_l", " ADA_L ", "ＡＤＡ_Ｌ"} {
		s.Equal(ada.String(), s.found(name), "%q", name)
	}
	s.Equal(named.String(), s.found("STRASSE"))
}

func (s *testSuite) TestAGuestNameFindsItsAccount() {
	s.Equal(guest.String(), s.found("guest_a1b2c3"))
	s.Equal(guest.String(), s.found(" guest_a1b2c3 "))
}

func (s *testSuite) TestAnAccountThatTookAUsernameIsStillFoundUnderItsGuestName() {
	s.Equal(named.String(), s.found("guest_d4e5f6"))
}

func (s *testSuite) TestANameNoAccountIsShownUnderFindsNobody() {
	for _, name := range []string{"", "   ", "Bob", "guest_", "guest_ffffff", "GUEST_a1b2c3", "a1b2c3", "Ada!"} {
		_, err := s.query.Account(s.T().Context(), name)

		s.ErrorIs(err, account_query.ErrNoAccount, "%q", name)
	}
}

func (s *testSuite) TestTheNameAnAccountIsShownUnderFindsThatAccount() {
	shown, err := authors_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), cptime.NewFixedClock(today)).
		Authors(s.T().Context(), []players.AccountID{ada, guest, named})
	s.Require().NoError(err)
	s.Require().Len(shown.GetAuthors(), 3)

	for _, author := range shown.GetAuthors() {
		s.Equal(author.GetAccountId(), s.found(author.GetName()), "%q", author.GetName())
	}
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query.Account(ctx, "Ada_L")

	s.Require().Error(err)
	s.NotErrorIs(err, account_query.ErrNoAccount, "not a name nobody holds")
}
