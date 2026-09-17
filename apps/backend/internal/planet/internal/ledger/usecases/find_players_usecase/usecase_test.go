package find_players_usecase_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/find_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) { return o[tile], true }

type borders map[uint32]string

func (b borders) CountryOf(tile uint32) string { return b[tile] }

type bans map[string]antibot.Sentence

func (b bans) Sentence(scope string) (antibot.Sentence, bool) {
	sentence, ok := b[scope]
	return sentence, ok
}

type countries struct{}

func (countries) CheckCountry(country string) bool { return len(country) == 2 }

// Tiles 1-4 are Israel's ground, 5 is Jordan's. Every take is a second apart, in order.
func setup(t *testing.T) (*inmemory_ledger_storage.Storage, owners) {
	t.Helper()

	clock := cptime.NewFixedClock(start)
	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	current := owners{}

	take := func(tile uint32, scope, country string) {
		book.Append(ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: current[tile], At: clock.Now()})
		current[tile] = country
		clock.Advance(time.Second)
	}

	take(1, "bot", "ps")
	take(2, "player", "ps")
	take(3, "bot", "ps")
	take(5, "far", "ps")
	take(4, "covered", "ps")
	take(4, "defender", "il")

	return book, current
}

var israel = borders{1: "il", 2: "il", 3: "il", 4: "il", 5: "jo"}

func TestItListsEveryoneWhoPaintedTheFlagInTheAreaLatestFirst(t *testing.T) {
	book, current := setup(t)
	useCase := find_players_usecase.New(book, current, israel, bans{"bot": {Offence: 2, Until: start.Add(time.Hour)}}, countries{})

	out, err := useCase.Execute(t.Context(), find_players_usecase.In{Flag: "ps", Area: "il"})
	require.NoError(t, err)

	assert.Equal(t, find_players_usecase.Out{Total: 3, Players: []ledger.Player{
		{Scope: "covered", Takes: 1, FirstAt: start.Add(4 * time.Second), LastAt: start.Add(4 * time.Second)},
		{
			Scope: "bot", Tiles: 2, Takes: 2, FirstAt: start, LastAt: start.Add(2 * time.Second),
			Banned: true, BannedUntil: start.Add(time.Hour), Offence: 2,
		},
		{Scope: "player", Tiles: 1, Takes: 1, FirstAt: start.Add(time.Second), LastAt: start.Add(time.Second)},
	}}, out, "a tile outside the area is nobody's, and a tile taken back still counts as a take")
}

func TestNoAreaIsTheWholeMapAndTheLimitCuts(t *testing.T) {
	book, current := setup(t)
	useCase := find_players_usecase.New(book, current, israel, nil, countries{})

	out, err := useCase.Execute(t.Context(), find_players_usecase.In{Flag: "ps", Limit: 1})
	require.NoError(t, err)

	assert.Equal(t, 4, out.Total)
	require.Len(t, out.Players, 1)
	assert.Equal(t, "covered", out.Players[0].Scope)
}

func TestItRefusesAnUnknownFlagOrArea(t *testing.T) {
	book, current := setup(t)
	useCase := find_players_usecase.New(book, current, israel, nil, countries{})

	_, err := useCase.Execute(t.Context(), find_players_usecase.In{Flag: ""})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)

	_, err = useCase.Execute(t.Context(), find_players_usecase.In{Flag: "ps", Area: "israel"})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)
}
