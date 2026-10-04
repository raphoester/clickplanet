package e2e_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func currentSeason(t *testing.T, database cppg.Config) seasons.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "seasons.yaml")
	require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `
seasons:
  list:
    - number: 0
      endsAt: %s
      finale: 2h
`, time.Now().Add(30*24*time.Hour).UTC().Format(time.RFC3339)), 0o600))
	var config struct{ Seasons seasons.Config }
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))
	config.Seasons.Database = database

	return config.Seasons
}

func (p *gamer) seasons() seasonsv1connect.SeasonServiceClient {
	return seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) mySeason() *seasonsv1.GetMySeasonResponse {
	p.t.Helper()

	req := connect.NewRequest(&seasonsv1.GetMySeasonRequest{})
	p.send(req.Header())
	res, err := p.seasons().GetMySeason(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func (p *gamer) worn() *playerv1.Title {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetTitlesRequest{})
	p.send(req.Header())
	res, err := p.players().GetTitles(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetWorn()
}

func (s gameStack) standings(t *testing.T, country string) []*seasonsv1.Standing {
	t.Helper()

	res, err := seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, s.baseURL, connect.WithHTTPGet()).
		GetStandings(t.Context(), connect.NewRequest(&seasonsv1.GetStandingsRequest{CountryId: country}))
	require.NoError(t, err)
	assert.Equal(t, "public, max-age=15", res.Header().Get("Cache-Control"))
	return res.Msg.GetStandings()
}

func (s gameStack) namedPlayer(t *testing.T, subject, name string) *gamer {
	t.Helper()

	named := s.newPlayer(t)
	named.link(subject)
	_, err := named.setName(name)
	require.NoError(t, err)
	return named
}

func TestPlayersWithAUsernameAreRankedByTheTilesTheyTakeForTheirMainFlag(t *testing.T) {
	game := startGame(t)
	ada := game.namedPlayer(t, "google-ada", "Ada")
	require.NoError(t, ada.setColor(playerv1.NameColor_NAME_COLOR_PINK))
	cyd := game.namedPlayer(t, "google-cyd", "Cyd")

	ada.click(1, "fr")
	ada.click(2, "fr")
	ada.click(3, "de")
	cyd.click(4, "fr")

	require.Eventually(t, func() bool { return ada.mySeason().GetTiles() == 2 && cyd.mySeason().GetTiles() == 1 },
		5*time.Second, 20*time.Millisecond)
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		mine := ada.mySeason()
		assert.True(c, proto.Equal(&seasonsv1.GetMySeasonResponse{
			CountryId: "fr", Tiles: 2, GlobalRank: 1, CountryRank: 1, WornTitle: ada.worn(),
		}, mine), mine)
		mine = cyd.mySeason()
		assert.True(c, proto.Equal(&seasonsv1.GetMySeasonResponse{
			CountryId: "fr", Tiles: 1, GlobalRank: 2, CountryRank: 2, WornTitle: cyd.worn(),
		}, mine), mine)
	}, 5*time.Second, 20*time.Millisecond, "a title the take earns is granted apart from the tally")

	for _, country := range []string{"", "fr"} {
		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			standings := game.standings(t, country)
			top := []*seasonsv1.Standing{
				{Rank: 1, Name: "Ada", Color: playerv1.NameColor_NAME_COLOR_PINK, CountryId: "fr", Tiles: 2, WornTitle: ada.worn()},
				{Rank: 2, Name: "Cyd", CountryId: "fr", Tiles: 1, WornTitle: cyd.worn()},
			}
			require.Len(c, standings, len(top), country)
			for i := range top {
				assert.True(c, proto.Equal(top[i], standings[i]), standings[i])
			}
		}, 5*time.Second, 20*time.Millisecond, country)
	}
	assert.Empty(t, game.standings(t, "de"))
}

func TestAGuestReadsItsTilesAndIsNeitherRankedNorListed(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)
	require.NoError(t, guest.announce("de"))

	guest.click(5, "de")

	require.Eventually(t, func() bool { return guest.mySeason().GetTiles() == 1 }, 5*time.Second, 20*time.Millisecond)
	assert.True(t, proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "de", Tiles: 1}, guest.mySeason()), guest.mySeason())
	assert.Empty(t, game.standings(t, ""))
}

func TestTheCallersSeasonWithNoTokenIsUnauthenticated(t *testing.T) {
	game := startGame(t)

	_, err := seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, game.baseURL).
		GetMySeason(t.Context(), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestTheCallersSeasonReadsAsTheIdentityTheCookieResumes(t *testing.T) {
	game := startGame(t)
	ada := game.namedPlayer(t, "google-ada", "Ada")
	ada.click(1, "fr")
	require.Eventually(t, func() bool { return ada.mySeason().GetTiles() == 1 }, 5*time.Second, 20*time.Millisecond)

	back := ada.resumed()

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		mine := back.mySeason()
		assert.True(c, proto.Equal(&seasonsv1.GetMySeasonResponse{
			CountryId: "fr", Tiles: 1, GlobalRank: 1, CountryRank: 1, WornTitle: ada.worn(),
		}, mine), mine)
	}, 5*time.Second, 20*time.Millisecond, "a title the take earns is granted apart from the tally")
}

func TestARebuildCountsTheStandingsAgainFromTheLogWithoutWhatWasReverted(t *testing.T) {
	game := startGame(t)
	ada := game.namedPlayer(t, "google-ada", "Ada")
	bob := game.namedPlayer(t, "google-bob", "Bob")
	ada.click(1, "fr")
	ada.click(2, "fr")
	bob.click(3, "de")
	require.Eventually(t, func() bool { return ada.mySeason().GetTiles() == 2 && bob.mySeason().GetTiles() == 1 },
		5*time.Second, 20*time.Millisecond)

	_, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL).RevertPlayer(t.Context(),
		connect.NewRequest(&planetv1.RevertPlayerRequest{AccountId: bob.account().String()}))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var reverted int
		err := game.schema(t, "planet").QueryRowContext(t.Context(), `SELECT count(*) FROM ledger_takes WHERE reverted`).Scan(&reverted)
		return err == nil && reverted == 1
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, uint64(1), bob.mySeason().GetTiles(), "the live count keeps what it counted")

	_, err = seasonsv1connect.NewAdminServiceClient(http.DefaultClient, game.baseURL).
		RebuildStandings(t.Context(), connect.NewRequest(&seasonsv1.RebuildStandingsRequest{}))
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err), "the public router does not serve it")
	rebuilt, err := seasonsv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL).
		RebuildStandings(t.Context(), connect.NewRequest(&seasonsv1.RebuildStandingsRequest{}))
	require.NoError(t, err)
	assert.Zero(t, rebuilt.Msg.GetFromPosition())

	require.Eventually(t, func() bool { return bob.mySeason().GetTiles() == 0 && ada.mySeason().GetTiles() == 2 },
		5*time.Second, 20*time.Millisecond, "every take counted again but the one reverted")
}
