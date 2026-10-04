package authors_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler/authors_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/postgres_worn_title_store"
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
	titles  *postgres_title_store.Store
	worn    *postgres_worn_title_store.Store
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
	s.titles = postgres_title_store.New(s.db)
	s.worn = postgres_worn_title_store.New(s.db)
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

func (s *testSuite) query() *authors_query.PostgresQuery {
	return authors_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), cptime.NewFixedClock(today))
}

func (s *testSuite) authors(accounts ...players.AccountID) map[string]*playerv1.Author {
	answer, err := s.query().Authors(s.T().Context(), accounts)
	s.Require().NoError(err)

	by := make(map[string]*playerv1.Author, len(answer.GetAuthors()))
	for _, author := range answer.GetAuthors() {
		by[author.GetAccountId()] = author
	}
	return by
}

func names(authors map[string]*playerv1.Author) map[string]string {
	by := make(map[string]string, len(authors))
	for account, author := range authors {
		by[account] = author.GetName()
	}
	return by
}

func (s *testSuite) TestAPlayerIsNamedByItsUsernameAndAGuestByItsCode() {
	s.Equal(map[string]string{ada.String(): "Ada", guest.String(): "guest_a1b2c3"}, names(s.authors(ada, guest)))
}

func (s *testSuite) TestAUsernameWinsOverTheGuestCodeItNoLongerGoesBy() {
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), ada, "91aa3d"))

	s.Equal(map[string]string{ada.String(): "Ada"}, names(s.authors(ada)))
}

func (s *testSuite) TestAGuestIsToldApartFromAUsername() {
	authors := s.authors(ada, guest)

	s.False(authors[ada.String()].GetGuest())
	s.True(authors[guest.String()].GetGuest())
}

func (s *testSuite) TestAnAdminCarriesItsMark() {
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	authors := s.authors(ada, bob)

	s.True(authors[ada.String()].GetAdmin())
	s.False(authors[bob.String()].GetAdmin())
}

func (s *testSuite) TestAnAccountNobodyCanNameIsLeftOut() {
	unnamed := players.AccountID{15: 4}
	s.Require().NoError(s.players.RecordTake(s.T().Context(), unnamed, today))

	s.Equal(map[string]string{ada.String(): "Ada"}, names(s.authors(ada, unnamed, players.AccountID{15: 5})),
		"an account with neither a profile nor a code is absent, and so is an unknown one")
}

func (s *testSuite) TestAskingGivesNobodyAGuestCode() {
	unnamed := players.AccountID{15: 4}

	s.authors(unnamed)

	_, err := s.players.GuestCode(s.T().Context(), unnamed)
	s.ErrorIs(err, players.ErrNoGuestCode, "a read path never writes")
}

func (s *testSuite) TestADeletedAccountCanNoLongerBeNamed() {
	s.Require().NoError(s.players.DeleteAccount(s.T().Context(), ada))

	s.Empty(s.authors(ada), "what a caller shows in its place is the caller's to decide")
}

func (s *testSuite) TestAskingAboutNobodyAnswersNobody() {
	s.Empty(s.authors())
}

func (s *testSuite) TestARepeatedAccountIsAnsweredOnceInTheOrderAsked() {
	answer, err := s.query().Authors(s.T().Context(), []players.AccountID{guest, ada, guest, bob})
	s.Require().NoError(err)

	ids := make([]string, 0, len(answer.GetAuthors()))
	for _, author := range answer.GetAuthors() {
		ids = append(ids, author.GetAccountId())
	}
	s.Equal([]string{guest.String(), ada.String(), bob.String()}, ids)
}

func (s *testSuite) TestEachAuthorCarriesItsColorAndItsStreakAsOfToday() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_PINK)))
	for _, account := range []players.AccountID{ada, guest} {
		s.Require().NoError(s.players.RecordTake(s.T().Context(), account, today.AddDate(0, 0, -1)))
		s.Require().NoError(s.players.RecordTake(s.T().Context(), account, today))
	}
	s.Require().NoError(s.players.RecordTake(s.T().Context(), bob, today.AddDate(0, 0, -3)))

	authors := s.authors(ada, bob, guest)

	s.Equal(playerv1.NameColor_NAME_COLOR_PINK, authors[ada.String()].GetColor())
	s.Equal(uint32(2), authors[ada.String()].GetStreak())
	s.Zero(authors[bob.String()].GetStreak(), "a streak broken since reads zero")
	s.Equal(playerv1.NameColor_NAME_COLOR_UNSPECIFIED, authors[guest.String()].GetColor())
	s.Zero(authors[guest.String()].GetStreak(), "a guest shows no streak, however long it runs")
}

func (s *testSuite) TestAnAuthorWhoOnlyPostedHasNoStreak() {
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), ada))

	s.Zero(s.authors(ada)[ada.String()].GetStreak())
}

func (s *testSuite) TestEachAuthorWearsItsTitleAndAGuestNone() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"og", "settler"}, bob: {"og", "settler"}, guest: {"og"}}, today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "og", today))

	authors := s.authors(ada, bob, guest)

	s.Equal("og", authors[ada.String()].GetWornTitle().GetId())
	s.Equal("OG", authors[ada.String()].GetWornTitle().GetName())
	s.Equal("settler", authors[bob.String()].GetWornTitle().GetId(), "no choice wears the first rank shown")
	s.Nil(authors[guest.String()].GetWornTitle())
}

func (s *testSuite) TestAColorTheProtoDoesNotNameIsAnError() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, 99))

	_, err := s.query().Authors(s.T().Context(), []players.AccountID{ada})

	s.ErrorIs(err, playerread.ErrUnknownColor)
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Authors(ctx, []players.AccountID{ada})

	s.Error(err)
}

func (s *testSuite) TestEachAuthorIsWhatGetAuthorAnswersForIt() {
	cy, dan, eve := players.AccountID{15: 4}, players.AccountID{15: 5}, players.AccountID{15: 6}
	s.named(cy, "Cyd")
	s.named(dan, "Dan")
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), eve, "0d0e0f"))
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_PINK)))
	takes := map[players.AccountID][]time.Time{
		ada:   {today.AddDate(0, 0, -1), today},
		bob:   {today.AddDate(0, 0, -3), today.AddDate(0, 0, -2)},
		guest: {today.AddDate(0, 0, -1), today},
		dan:   {today.AddDate(0, 0, 1)},
	}
	for account, at := range takes {
		for _, take := range at {
			s.Require().NoError(s.players.RecordTake(s.T().Context(), account, take))
		}
	}
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), cy))
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{
		ada: {"og", "settler"}, bob: {"settler", "raider", "loyal"}, guest: {"og"}, dan: {"retired", "og"},
	}, today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "og", today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), bob, "settler", today))
	s.Require().NoError(s.worn.Wear(s.T().Context(), guest, "og", today))

	catalog := titles.NewCatalog()
	single := get_author_usecase.New(s.players, players.NewGuestCodes(s.players, &players.SequentialCodes{}),
		wearing.NewWardrobe(s.worn, titles.NewBook(s.titles, catalog), catalog), cptime.NewFixedClock(today))
	everyone := []players.AccountID{ada, bob, guest, cy, dan, eve}
	many := s.authors(everyone...)

	s.Require().Len(many, len(everyone))
	for _, account := range everyone {
		one, err := single.Execute(s.T().Context(), account)
		s.Require().NoError(err)

		s.True(proto.Equal(&playerv1.Author{
			AccountId: account.String(),
			Name:      one.Name(),
			Admin:     one.Admin(),
			Color:     playermessage.Color(one.Color()),
			Streak:    one.Streak().Days(),
			WornTitle: playermessage.Title(one.Worn()),
			Guest:     one.Guest(),
		}, many[account.String()]), "%s: GetAuthor and GetAuthors must name an account alike, got %v", one.Name(), many[account.String()])
	}
}
