package matches_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler/authors_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_handler/matches_query"
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
}

var (
	ada     = players.AccountID{15: 1}
	eire    = players.AccountID{15: 2}
	eire1   = players.AccountID{15: 3}
	iraEire = players.AccountID{15: 4}
	guest   = players.AccountID{15: 5}
	today   = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.named(ada, "Ada")
	s.named(iraEire, "IRA EIRE")
	s.named(eire1, "EIRE1")
	s.named(eire, "EIRE")
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), ada, "91aa3d"))
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, today)))
}

func (s *testSuite) search(folded string, limit int) *playerv1.SearchPlayersResponse {
	answer, err := matches_query.NewPostgresQuery(s.db).Search(s.T().Context(), folded, limit)
	s.Require().NoError(err)
	return answer
}

func names(answer *playerv1.SearchPlayersResponse) []string {
	out := make([]string, 0, len(answer.GetPlayers()))
	for _, match := range answer.GetPlayers() {
		out = append(out, match.GetName())
	}
	return out
}

func (s *testSuite) TestAPartOfANameFindsTheSameNameFirstThenTheNamesItStarts() {
	answer := s.search("eire", 20)
	s.Equal([]string{"EIRE", "EIRE1", "IRA EIRE"}, names(answer))
	s.Equal(uint32(3), answer.GetTotal())
}

func (s *testSuite) TestTheLimitCutsAndTheTotalCountsEveryAccountFound() {
	answer := s.search("eire", 1)
	s.Equal([]string{"EIRE"}, names(answer))
	s.Equal(uint32(3), answer.GetTotal())
}

func (s *testSuite) TestAGuestIsFoundByItsCodeUntilItChoosesAUsername() {
	answer := s.search("guest_a1", 20)
	s.Require().Len(answer.GetPlayers(), 1)
	s.Equal(guest.String(), answer.GetPlayers()[0].GetAccountId())
	s.True(answer.GetPlayers()[0].GetGuest())

	answer = s.search("91aa", 20)
	s.Empty(answer.GetPlayers(), "the chat and the back office name Ada by its username, not the code it went by")
	s.Zero(answer.GetTotal())
}

func (s *testSuite) TestADeletedAccountIsNotFound() {
	s.Require().NoError(s.players.DeleteAccount(s.T().Context(), eire))
	s.Equal([]string{"EIRE1", "IRA EIRE"}, names(s.search("eire", 20)))
}

func (s *testSuite) TestEachMatchIsNamedAsGetAuthorsNamesIt() {
	answer := s.search("a", 20)
	s.Require().Len(answer.GetPlayers(), 3)

	authors, err := authors_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), cptime.NewFixedClock(today)).
		Authors(s.T().Context(), []players.AccountID{ada, guest, iraEire})
	s.Require().NoError(err)
	named := make(map[string]*playerv1.Author, len(authors.GetAuthors()))
	for _, author := range authors.GetAuthors() {
		named[author.GetAccountId()] = author
	}

	for _, match := range answer.GetPlayers() {
		author := named[match.GetAccountId()]
		s.Require().NotNil(author, match.GetName())
		s.Equal(author.GetName(), match.GetName())
		s.Equal(author.GetGuest(), match.GetGuest(), match.GetName())
	}
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()
	_, err := matches_query.NewPostgresQuery(s.db).Search(ctx, "eire", 20)
	s.Error(err)
}
