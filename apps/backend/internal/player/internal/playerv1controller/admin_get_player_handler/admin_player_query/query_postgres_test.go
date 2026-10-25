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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
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
	kept map[players.AccountID]players.Account
	err  error
}

func (f *fakeAccounts) Account(_ context.Context, account players.AccountID) (players.Account, error) {
	return f.kept[account], f.err
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
	guest     = players.AccountID{15: 2}
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
	s.accounts = &fakeAccounts{kept: map[players.AccountID]players.Account{
		ada:   players.AccountOf(true, createdAt),
		guest: players.AccountOf(false, createdAt),
	}}
	s.fronts = &fakeFronts{answer: &playerv1.GetFrontsResponse{}}
	s.clock = cptime.NewFixedClock(monday)
	s.Require().NoError(s.players.SaveProfile(s.T().Context(), players.NewProfile(ada, "Ada_L", monday)))
}

func (s *testSuite) query() *admin_player_query.PostgresQuery {
	return admin_player_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), s.accounts, s.fronts, s.clock)
}

func (s *testSuite) player(account players.AccountID) *playerv1.AdminGetPlayerResponse {
	answer, err := s.query().Player(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestAUsernameIsReadWithEverythingTheGameKeepsAboutIt() {
	ctx := s.T().Context()
	s.Require().NoError(s.players.Record(ctx, ada, players.TakeAt(0, monday.Add(-24*time.Hour))))
	s.Require().NoError(s.players.Record(ctx, ada, players.TakeAt(1, monday)))
	s.Require().NoError(s.players.Record(ctx, ada, players.MessageAt(monday)))
	s.Require().NoError(s.players.SaveColor(ctx, ada, players.Color(playerv1.NameColor_NAME_COLOR_TEAL)))
	_, err := s.db.On(ctx).ExecContext(ctx, `UPDATE profiles SET admin = true WHERE account_id = $1`, uuid.UUID(ada))
	s.Require().NoError(err)
	s.Require().NoError(s.titles.Grant(ctx, titles.Holdings{ada: {"settler", "og"}}, monday))
	s.Require().NoError(s.worn.Wear(ctx, ada, "og", monday))

	og, _ := titles.NewCatalog().StandingOf("og")
	settler, _ := titles.NewCatalog().StandingOf("settler")
	s.True(proto.Equal(&playerv1.AdminGetPlayerResponse{
		Name:            "Ada_L",
		Linked:          true,
		Admin:           true,
		Color:           playerv1.NameColor_NAME_COLOR_TEAL,
		CreatedAtUnixMs: createdAt.UnixMilli(),
		Stats:           &playerv1.Stats{TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: "2026-09-14"},
		MessagesSent:    1,
		Titles:          playermessage.Titles([]titles.Standing{og, settler}),
		WornTitle:       playermessage.Title(og),
	}, s.player(ada)), "%v", s.player(ada))
}

func (s *testSuite) TestAGuestIsReadUnderItsCodeWithTheStreakItKeeps() {
	s.Require().NoError(s.players.SaveGuestCode(s.T().Context(), guest, "a1b2c3"))
	s.Require().NoError(s.players.Record(s.T().Context(), guest, players.TakeAt(2, monday)))

	player := s.player(guest)

	s.Equal("guest_a1b2c3", player.GetName())
	s.True(player.GetGuest())
	s.False(player.GetLinked())
	s.Equal(createdAt.UnixMilli(), player.GetCreatedAtUnixMs())
	s.Equal(uint32(1), player.GetStats().GetStreakCurrent(), "the game shows a guest no streak, and the operator what is kept")
}

func (s *testSuite) TestAnAccountThatOnlyTookTilesHasNoName() {
	s.Require().NoError(s.players.Record(s.T().Context(), guest, players.TakeAt(3, monday)))

	player := s.player(guest)

	s.Empty(player.GetName(), "it never announced and never posted, so it was never given a code")
	s.True(player.GetGuest())
	s.Equal(uint64(1), player.GetStats().GetTilesTaken())
}

func (s *testSuite) TestAnAccountTheGameKeepsNothingAboutIsNoPlayer() {
	_, err := s.query().Player(s.T().Context(), players.AccountID{15: 9})

	s.ErrorIs(err, admin_player_query.ErrNoPlayer)
}

func (s *testSuite) TestEveryCountryThePlayerPlaysForAndAgainstIsRead() {
	countries := func(codes ...string) []*playerv1.CountryTiles {
		tiles := make([]*playerv1.CountryTiles, 0, len(codes))
		for i, code := range codes {
			tiles = append(tiles, &playerv1.CountryTiles{CountryId: code, Tiles: uint64(10 - i)}) //nolint:gosec // a few countries.
		}
		return tiles
	}
	s.fronts.answer = &playerv1.GetFrontsResponse{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it", "pt")}

	player := s.player(ada)

	s.True(proto.Equal(&playerv1.AdminGetPlayerResponse{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it", "pt")},
		&playerv1.AdminGetPlayerResponse{PlaysFor: player.GetPlaysFor(), PlaysAgainst: player.GetPlaysAgainst()}),
		"not the public card's top three: %v", player)
}

func (s *testSuite) TestAnAccountAuthNoLongerKnowsHasNoCreationDate() {
	delete(s.accounts.kept, ada)

	player := s.player(ada)

	s.Zero(player.GetCreatedAtUnixMs())
	s.False(player.GetLinked())
}

func (s *testSuite) TestAFailureToAskAuthOrPlanetIsAnError() {
	for _, fail := range []func(error){
		func(err error) { s.accounts.err = err },
		func(err error) { s.fronts.err = err },
	} {
		s.SetupTest()
		cause := errors.New("down")
		fail(cause)

		_, err := s.query().Player(s.T().Context(), ada)

		s.Require().ErrorIs(err, cause)
		s.NotErrorIs(err, admin_player_query.ErrNoPlayer)
	}
}

func (s *testSuite) TestAStoreFailureIsNotNoPlayer() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Player(ctx, ada)

	s.Require().Error(err)
	s.NotErrorIs(err, admin_player_query.ErrNoPlayer)
}
