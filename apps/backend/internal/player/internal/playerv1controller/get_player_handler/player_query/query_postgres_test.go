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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/postgres_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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
	titles   *postgres_title_store.Store
	worn     *postgres_worn_title_store.Store
	fronts   *postgres_front_store.Store
	accounts *fakeAccounts
	clock    *cptime.FixedClock
}

var (
	ada       = players.AccountID{15: 1}
	monday    = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	createdAt = time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
	s.titles = postgres_title_store.New(s.db)
	s.worn = postgres_worn_title_store.New(s.db)
	s.fronts = postgres_front_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.accounts = &fakeAccounts{created: map[players.AccountID]time.Time{ada: createdAt}}
	s.clock = cptime.NewFixedClock(monday)
	s.named(ada, "Ada_L")
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, monday)))
}

func (s *testSuite) query() *player_query.PostgresQuery {
	return player_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), s.accounts, s.clock)
}

func (s *testSuite) player(name string) *playerv1.Player {
	answer, err := s.query().Player(s.T().Context(), name)
	s.Require().NoError(err)
	return answer.GetPlayer()
}

func (s *testSuite) TestAPlayerIsFoundByItsNameInAnyCase() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday.Add(-24*time.Hour)))
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))

	s.True(proto.Equal(&playerv1.Player{
		Name:            "Ada_L",
		Stats:           &playerv1.Stats{TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: "2026-09-14"},
		CreatedAtUnixMs: createdAt.UnixMilli(),
		Titles:          []*playerv1.Title{},
	}, s.player("aDA_l")))
}

func (s *testSuite) TestANameOfAnyScriptIsFoundByItsFold() {
	for i, pair := range [][2]string{{"Straße", "STRASSE"}, {"Ａｄａ", "ada"}, {"Жанна", "жАННА"}} {
		s.named(players.AccountID{14: 1, 15: byte(i)}, players.Name(pair[0]))

		s.Equal(pair[0], s.player(pair[1]).GetName(), "%q finds %q", pair[1], pair[0])
	}
}

func (s *testSuite) TestAnAdminIsSaidToBeOne() {
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	s.True(s.player("Ada_L").GetAdmin())
}

func (s *testSuite) TestThePlayerCarriesItsColor() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))

	s.Equal(playerv1.NameColor_NAME_COLOR_TEAL, s.player("Ada_L").GetColor())
}

func (s *testSuite) TestThePlayerShowsItsBestOfEachTrackAndWearsTheFirst() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"loyal", "raider", "settler", "og"}}, monday))

	player := s.player("Ada_L")

	raider, _ := titles.NewCatalog().StandingOf("raider")
	loyal, _ := titles.NewCatalog().StandingOf("loyal")
	s.True(proto.Equal(playermessage.Title(raider), player.GetWornTitle()))
	og, _ := titles.NewCatalog().StandingOf("og")
	want := playermessage.Titles([]titles.Standing{og, raider, loyal})
	s.Require().Len(player.GetTitles(), len(want))
	for i := range want {
		s.True(proto.Equal(want[i], player.GetTitles()[i]), want[i].GetId())
	}
}

func (s *testSuite) TestThePlayerWearsTheTitleItChose() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"loyal", "settler", "og"}}, monday))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "loyal", monday))

	s.Equal("loyal", s.player("Ada_L").GetWornTitle().GetId())
}

func (s *testSuite) TestATitleTheCatalogNoLongerHasIsNotShown() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"retired", "og"}}, monday))

	shown := s.player("Ada_L").GetTitles()

	s.Require().Len(shown, 1, "the next reconciliation revokes it")
	s.Equal("og", shown[0].GetId())
}

func (s *testSuite) took(account players.AccountID, country, previous fronts.Country, tiles int) {
	take, err := fronts.NewTake(account, country, previous)
	s.Require().NoError(err)
	for range tiles {
		s.Require().NoError(s.fronts.RecordTake(s.T().Context(), take))
	}
}

func (s *testSuite) TestThePlayerPlaysForAndAgainstEachCountryMostTilesFirst() {
	s.took(ada, "fr", "de", 3)
	s.took(ada, "fr", "", 2)
	s.took(ada, "it", "es", 1)
	s.took(ada, "be", "it", 1)
	s.took(players.AccountID{15: 2}, "de", "fr", 9)

	player := s.player("Ada_L")

	s.True(proto.Equal(&playerv1.Player{
		PlaysFor: []*playerv1.CountryTiles{{CountryId: "fr", Tiles: 5}, {CountryId: "be", Tiles: 1}, {CountryId: "it", Tiles: 1}},
		PlaysAgainst: []*playerv1.CountryTiles{
			{CountryId: "de", Tiles: 3}, {CountryId: "es", Tiles: 1}, {CountryId: "it", Tiles: 1},
		},
	}, &playerv1.Player{PlaysFor: player.GetPlaysFor(), PlaysAgainst: player.GetPlaysAgainst()}))
}

func (s *testSuite) TestAPlayerThatTookNothingPlaysForAndAgainstNobody() {
	player := s.player("Ada_L")

	s.Empty(player.GetPlaysFor())
	s.Empty(player.GetPlaysAgainst())
}

func (s *testSuite) TestTheStreakIsReadAsOfToday() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))
	s.clock.Advance(48 * time.Hour)

	stats := s.player("Ada_L").GetStats()

	s.Zero(stats.GetStreakCurrent())
	s.Equal(uint32(1), stats.GetStreakBest())
}

func (s *testSuite) TestANameNobodyHoldsIsNoProfile() {
	s.named(ada, "Ada_Lovelace")

	for _, name := range []string{"Bob", "Ada_L"} {
		_, err := s.query().Player(s.T().Context(), name)

		s.ErrorIs(err, player_query.ErrNoPlayer, "%q: a rename frees the old name", name)
	}
}

func (s *testSuite) TestANameNoAccountMayHoldFindsNobody() {
	for _, name := range []string{"", "guest_Ada", "a  b", "Ada!"} {
		_, err := s.query().Player(s.T().Context(), name)

		s.ErrorIs(err, player_query.ErrNoPlayer, "%q", name)
	}
}

func (s *testSuite) TestAnAccountAuthNoLongerKnowsHasNoCreationDate() {
	s.named(players.AccountID{15: 2}, "Bob")

	s.Zero(s.player("Bob").GetCreatedAtUnixMs())
}

func (s *testSuite) TestAColorTheProtoDoesNotNameIsAnError() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, 99))

	_, err := s.query().Player(s.T().Context(), "Ada_L")

	s.ErrorIs(err, playerread.ErrUnknownColor)
}

func (s *testSuite) TestAStoreFailureIsNotNoProfile() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Player(ctx, "Ada_L")

	s.Require().Error(err)
	s.NotErrorIs(err, player_query.ErrNoPlayer)
}

func (s *testSuite) TestAnAuthFailureIsAnError() {
	s.accounts.err = errors.New("auth is down")

	_, err := s.query().Player(s.T().Context(), "Ada_L")

	s.Require().ErrorIs(err, s.accounts.err)
	s.NotErrorIs(err, player_query.ErrNoPlayer)
}

func (s *testSuite) TestANameIsFoundExactlyWhenTheDomainFoldsItToANameKept() {
	kept := []players.Name{"Ada_L", "Straße", "Émile", "Ａｄａ", "Жанна", "東京タワー", "Ada L"}
	for i, name := range kept[1:] {
		s.named(players.AccountID{14: 2, 15: byte(i)}, name)
	}
	typed := []string{
		"Ada_L", "ada_l", " ADA_L ", "STRASSE", "strasse", "Strasse", "straße", " Straße  ", "E\u0301MILE", "émile",
		"EMILE", "ada", "ａｄａ", "ＡＤＡ", "жАННА", "ЖАННА", "東京タワー", " Ada L ", "ada l", "ada  l", "\u00a0Ada L",
		"A\u200bda L", "Ada\u00adL", "Ada!", "", "   ", "guest_ada", "ＧＵＥＳＴ_ada", "Ada_L\u0301", "ada_ｌ", "ＡＤＡ_Ｌ",
	}

	for _, value := range typed {
		var want players.Name
		if name, err := players.NameOf(value); err == nil {
			for _, k := range kept {
				if k.Folded() == name.Folded() {
					want = k
				}
			}
		}

		answer, err := s.query().Player(s.T().Context(), value)

		if want == "" {
			s.ErrorIs(err, player_query.ErrNoPlayer, "%q", value)
			continue
		}
		if s.NoError(err, "%q", value) {
			s.Equal(string(want), answer.GetPlayer().GetName(), "%q", value)
		}
	}
}
