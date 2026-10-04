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
	assert.True(t, proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 2, GlobalRank: 1, CountryRank: 1},
		ada.mySeason()), ada.mySeason())
	assert.True(t, proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 1, GlobalRank: 2, CountryRank: 2},
		cyd.mySeason()), cyd.mySeason())

	top := []*seasonsv1.Standing{
		{Rank: 1, Name: "Ada", Color: playerv1.NameColor_NAME_COLOR_PINK, CountryId: "fr", Tiles: 2},
		{Rank: 2, Name: "Cyd", CountryId: "fr", Tiles: 1},
	}
	for _, country := range []string{"", "fr"} {
		standings := game.standings(t, country)
		require.Len(t, standings, len(top), country)
		for i := range top {
			assert.True(t, proto.Equal(top[i], standings[i]), standings[i])
		}
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

	assert.True(t, proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 1, GlobalRank: 1, CountryRank: 1},
		back.mySeason()), back.mySeason())
}
