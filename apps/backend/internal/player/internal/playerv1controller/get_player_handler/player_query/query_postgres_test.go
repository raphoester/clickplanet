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

type fakeFronts struct {
	answer *playerv1.GetFrontsResponse
	err    error
}

func (f *fakeFronts) Fronts(context.Context, players.AccountID) (*playerv1.GetFrontsResponse, error) {
	return f.answer, f.err
}

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	players  *postgres_player_store.Store
	titles   *postgres_title_store.Store
	worn     *postgres_worn_title_store.Store
	fronts   *fakeFronts
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
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.accounts = &fakeAccounts{created: map[players.AccountID]time.Time{ada: createdAt}}
	s.fronts = &fakeFronts{answer: &playerv1.GetFrontsResponse{}}
	s.clock = cptime.NewFixedClock(monday)
	s.named(ada, "Ada_L")
}

func (s *testSuite) named(account players.AccountID, name players.Name) {
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(account, name, monday)))
}

func (s *testSuite) query() *player_query.PostgresQuery {
	return player_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), s.accounts, s.fronts, s.clock)
}

func (s *testSuite) player(account players.AccountID) *playerv1.Player {
	answer, err := s.query().Player(s.T().Context(), account)
	s.Require().NoError(err)
	return answer.GetPlayer()
}

func (s *testSuite) TestAPlayerIsFoundByItsAccount() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday.Add(-24*time.Hour)))
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))

	s.True(proto.Equal(&playerv1.Player{
		Name:            "Ada_L",
		Stats:           &playerv1.Stats{TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: "2026-09-14"},
		CreatedAtUnixMs: createdAt.UnixMilli(),
		Titles:          []*playerv1.Title{},
	}, s.player(ada)))
}

func (s *testSuite) TestAnAccountKeepsItsCardWhenAnotherTakesTheNameItLeft() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))
	bob := players.AccountID{15: 2}
	s.named(ada, "Ada_Lovelace")
	s.named(bob, "Ada_L")

	s.True(proto.Equal(&playerv1.Player{
		Name:            "Ada_Lovelace",
		Stats:           &playerv1.Stats{TilesTaken: 1, StreakCurrent: 1, StreakBest: 1, StreakLastDay: "2026-09-14"},
		CreatedAtUnixMs: createdAt.UnixMilli(),
		Titles:          []*playerv1.Title{},
	}, s.player(ada)))
	s.Equal("Ada_L", s.player(bob).GetName())
	s.Zero(s.player(bob).GetStats().GetTilesTaken())
}

func (s *testSuite) TestAnAccountWithNoUsernameIsNoPlayer() {
	guest := players.AccountID{15: 3}
	s.Require().NoError(s.players.RecordTake(s.T().Context(), guest, monday))
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))

	for _, account := range []players.AccountID{guest, {15: 4}} {
		_, err := s.query().Player(s.T().Context(), account)

		s.ErrorIs(err, player_query.ErrNoPlayer, "%s", account)
	}
}

func (s *testSuite) TestAnAdminIsSaidToBeOne() {
	_, err := s.db.ExecContext(s.T().Context(), `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)

	s.True(s.player(ada).GetAdmin())
}

func (s *testSuite) TestThePlayerCarriesItsColor() {
	s.Require().NoError(s.players.SaveColor(s.T().Context(), ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))

	s.Equal(playerv1.NameColor_NAME_COLOR_TEAL, s.player(ada).GetColor())
}

func (s *testSuite) TestThePlayerShowsItsBestOfEachTrackAndWearsTheFirst() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"loyal", "raider", "settler", "og"}}, monday))

	player := s.player(ada)

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

	s.Equal("loyal", s.player(ada).GetWornTitle().GetId())
}

func (s *testSuite) TestATitleTheCatalogNoLongerHasIsNotShown() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"retired", "og"}}, monday))

	shown := s.player(ada).GetTitles()

	s.Require().Len(shown, 1, "the next reconciliation revokes it")
	s.Equal("og", shown[0].GetId())
}

func (s *testSuite) TestAnybodyReadsTheTopThreeCountriesThePlayerPlaysForAndAgainst() {
	countries := func(codes ...string) []*playerv1.CountryTiles {
		tiles := make([]*playerv1.CountryTiles, 0, len(codes))
		for i, code := range codes {
			tiles = append(tiles, &playerv1.CountryTiles{CountryId: code, Tiles: uint64(10 - i)}) //nolint:gosec // a few countries.
		}
		return tiles
	}
	s.fronts.answer = &playerv1.GetFrontsResponse{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it", "pt")}

	player := s.player(ada)

	s.True(proto.Equal(&playerv1.Player{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it")},
		&playerv1.Player{PlaysFor: player.GetPlaysFor(), PlaysAgainst: player.GetPlaysAgainst()}), "%v", player)
}

func (s *testSuite) TestAPlayerThatTookNothingPlaysForAndAgainstNobody() {
	player := s.player(ada)

	s.Empty(player.GetPlaysFor())
	s.Empty(player.GetPlaysAgainst())
}

func (s *testSuite) TestAFailureToAskForTheCountriesIsAnError() {
	s.fronts.err = errors.New("planet is down")

	_, err := s.query().Player(s.T().Context(), ada)

	s.Require().ErrorIs(err, s.fronts.err)
	s.NotErrorIs(err, player_query.ErrNoPlayer)
}

func (s *testSuite) TestTheStreakIsReadAsOfToday() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))
	s.clock.Advance(48 * time.Hour)

	stats := s.player(ada).GetStats()

	s.Zero(stats.GetStreakCurrent())
	s.Equal(uint32(1), stats.GetStreakBest())
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
