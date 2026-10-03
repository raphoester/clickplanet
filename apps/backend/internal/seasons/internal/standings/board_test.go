package standings_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
)

type world struct {
	t       *testing.T
	store   *inmemory_contribution_store.Store
	players *standings.FakePlayers
}

func newWorld(t *testing.T) world {
	return world{t: t, store: inmemory_contribution_store.New(), players: standings.NewFakePlayers()}
}

func account(n int) standings.AccountID {
	return standings.AccountID{0: 0x01, 14: byte(n >> 8), 15: byte(n)}
}

func (w world) take(n int, country standings.Country, tiles int) {
	w.t.Helper()

	for range tiles {
		require.NoError(w.t, w.store.RecordTake(w.t.Context(), 0,
			standings.Take{Account: account(n), Country: country, At: time.Now()}))
	}
}

func (w world) player(n int, country standings.Country, tiles int) {
	w.t.Helper()

	w.take(n, country, tiles)
	w.players.Add(account(n), standings.Player{Name: fmt.Sprintf("player_%d", n), Color: standings.Color(n % 12)})
}

func (w world) guest(n int, country standings.Country, tiles int) {
	w.t.Helper()

	w.take(n, country, tiles)
	w.players.Add(account(n), standings.Player{Name: fmt.Sprintf("guest_%06x", n), Guest: true})
}

func (w world) board() standings.Board {
	return standings.NewBoard(w.store, w.players)
}

func (w world) top(country standings.Country) []standings.Standing {
	w.t.Helper()

	top, err := w.board().Top(w.t.Context(), 0, country)
	require.NoError(w.t, err)
	return top
}

func (w world) place(n int) standings.Place {
	w.t.Helper()

	place, err := w.board().Place(w.t.Context(), 0, account(n))
	require.NoError(w.t, err)
	return place
}

func ranksAndNames(top []standings.Standing) []string {
	lines := make([]string, len(top))
	for i, standing := range top {
		lines[i] = fmt.Sprintf("%d %s %s %d", standing.Rank, standing.Player.Name, standing.Line.Country, standing.Line.Tiles)
	}
	return lines
}

func TestTheTopIsTheTenBestPlayersWithAUsername(t *testing.T) {
	w := newWorld(t)
	for n := 1; n <= 12; n++ {
		w.player(n, "fr", 20-n)
	}
	w.guest(100, "fr", 30)
	w.guest(101, "de", 15)

	top := w.top("")

	require.Len(t, top, standings.Shown)
	assert.Equal(t, "1 player_1 fr 19", ranksAndNames(top)[0])
	assert.Equal(t, "10 player_10 fr 10", ranksAndNames(top)[9])
	assert.Equal(t, standings.Color(1), top[0].Player.Color)
}

func TestPlayersWithAsManyTilesShareARank(t *testing.T) {
	w := newWorld(t)
	w.player(1, "fr", 5)
	w.player(2, "de", 5)
	w.guest(3, "fr", 4)
	w.player(4, "it", 3)

	assert.Equal(t, []string{"1 player_1 fr 5", "1 player_2 de 5", "3 player_4 it 3"}, ranksAndNames(w.top("")))
}

func TestTheTopOfACountryIsThePlayersWhoseMainFlagItIs(t *testing.T) {
	w := newWorld(t)
	w.player(1, "de", 9)
	w.take(1, "fr", 8)
	w.player(2, "fr", 2)
	w.player(3, "fr", 4)

	assert.Equal(t, []string{"1 player_3 fr 4", "2 player_2 fr 2"}, ranksAndNames(w.top("fr")))
}

func TestAnAccountThePlayerModuleCannotNameIsNotRanked(t *testing.T) {
	w := newWorld(t)
	w.take(1, "fr", 9)
	w.player(2, "fr", 2)

	assert.Equal(t, []string{"1 player_2 fr 2"}, ranksAndNames(w.top("")))
	assert.Equal(t, standings.Place{Line: standings.Line{Account: account(1), Country: "fr", Tiles: 9}}, w.place(1))
}

func TestTheTopReadsPastAPageOfGuests(t *testing.T) {
	w := newWorld(t)
	for n := range 250 {
		w.guest(1000+n, "fr", 3)
	}
	w.player(1, "fr", 2)

	assert.Equal(t, []string{"1 player_1 fr 2"}, ranksAndNames(w.top("")))
	assert.Equal(t, 2, w.players.Asked())
}

func TestAnEmptySeasonHasAnEmptyTopAndAsksNobody(t *testing.T) {
	w := newWorld(t)

	assert.Empty(t, w.top(""))
	assert.Zero(t, w.players.Asked())
}

func TestThePlaceCountsThePlayersAboveGloballyAndInTheMainFlag(t *testing.T) {
	w := newWorld(t)
	w.player(1, "fr", 5)
	w.player(2, "fr", 7)
	w.player(3, "de", 9)
	w.guest(4, "fr", 8)
	w.player(5, "fr", 5)
	w.player(6, "de", 3)

	assert.Equal(t, standings.Place{
		Line:       standings.Line{Account: account(1), Country: "fr", Tiles: 5},
		GlobalRank: 3, CountryRank: 2,
	}, w.place(1))
	assert.Equal(t, uint32(3), w.place(5).GlobalRank, "a tie shares the rank")
}

func TestTheBestPlayerIsFirstEverywhere(t *testing.T) {
	w := newWorld(t)
	w.player(1, "fr", 5)
	w.player(2, "de", 3)

	assert.Equal(t, uint32(1), w.place(1).GlobalRank)
	assert.Equal(t, uint32(1), w.place(1).CountryRank)
	assert.Equal(t, uint32(1), w.place(2).CountryRank)
}

func TestThePlaceCountsPastAPage(t *testing.T) {
	w := newWorld(t)
	for n := range 250 {
		w.player(1000+n, "de", 3)
	}
	w.player(1, "fr", 2)

	place := w.place(1)

	assert.Equal(t, uint32(251), place.GlobalRank)
	assert.Equal(t, uint32(1), place.CountryRank)
}

func TestAGuestHasItsTilesAndNoRank(t *testing.T) {
	w := newWorld(t)
	w.guest(1, "fr", 4)
	w.take(1, "de", 1)

	assert.Equal(t, standings.Place{Line: standings.Line{Account: account(1), Country: "fr", Tiles: 4}}, w.place(1))
}

func TestAnAccountThatTookNothingHasNoPlace(t *testing.T) {
	w := newWorld(t)
	w.player(2, "fr", 4)

	assert.Equal(t, standings.Place{}, w.place(1))
}

func TestAFailureToAskThePlayersIsAnError(t *testing.T) {
	w := newWorld(t)
	w.player(1, "fr", 4)
	refused := errors.New("the player module is down")
	w.players.FailWith(refused)

	_, err := w.board().Top(t.Context(), 0, "")
	require.ErrorIs(t, err, refused)
	_, err = w.board().Place(t.Context(), 0, account(1))
	require.ErrorIs(t, err, refused)
}

func TestAFailureToReadTheStandingsIsAnError(t *testing.T) {
	w := newWorld(t)
	refused := errors.New("the database is down")
	w.store.FailWith(refused)

	_, err := w.board().Top(t.Context(), 0, "")
	require.ErrorIs(t, err, refused)
	_, err = w.board().Place(t.Context(), 0, account(1))
	require.ErrorIs(t, err, refused)
}
