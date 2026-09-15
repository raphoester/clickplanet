package top_players_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/top_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) { return o[tile], true }

type bans map[string]antibot.Sentence

func (b bans) Sentence(scope string) (antibot.Sentence, bool) {
	sentence, ok := b[scope]
	return sentence, ok
}

func setup(t *testing.T) (*inmemory_ledger_storage.Storage, owners) {
	t.Helper()

	clock := cptime.NewFixedClock(start)
	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)
	current := owners{}

	take := func(tile uint32, scope, country string) {
		book.Append(ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: current[tile], At: clock.Now()})
		current[tile] = country
		clock.Advance(time.Second)
	}

	take(1, "painter", "fr")
	take(2, "painter", "ps")
	take(3, "painter", "fr")
	take(4, "late", "de")
	take(5, "covered", "il")
	take(6, "covered", "il")
	take(5, "late", "de")
	take(7, "bombed", "br")
	take(8, "bombed", "br")
	delete(current, 7)
	delete(current, 8)

	return book, current
}

func TestItRanksByTakesThenTilesHeldOverEveryFlag(t *testing.T) {
	book, current := setup(t)
	useCase := top_players_usecase.New(book, current, bans{"painter": {Offence: 1, Until: start.Add(time.Hour)}})

	out, err := useCase.Execute(t.Context(), top_players_usecase.In{})
	require.NoError(t, err)

	assert.Equal(t, top_players_usecase.Out{Total: 4, Players: []ledger.Player{
		{
			Scope: "painter", Tiles: 3, Takes: 3, FirstAt: start, LastAt: start.Add(2 * time.Second),
			Banned: true, BannedUntil: start.Add(time.Hour), Offence: 1,
		},
		{Scope: "late", Tiles: 2, Takes: 2, FirstAt: start.Add(3 * time.Second), LastAt: start.Add(6 * time.Second)},
		{Scope: "covered", Tiles: 1, Takes: 2, FirstAt: start.Add(4 * time.Second), LastAt: start.Add(5 * time.Second)},
		{Scope: "bombed", Takes: 2, FirstAt: start.Add(7 * time.Second), LastAt: start.Add(8 * time.Second)},
	}}, out, "a tile taken over and a tile bombed count as takes, not as tiles")
}

func TestTheLimitCutsAfterTheRanking(t *testing.T) {
	book, current := setup(t)
	useCase := top_players_usecase.New(book, current, nil)

	out, err := useCase.Execute(t.Context(), top_players_usecase.In{Limit: 1})
	require.NoError(t, err)

	assert.Equal(t, 4, out.Total)
	require.Len(t, out.Players, 1)
	assert.Equal(t, "painter", out.Players[0].Scope)
	assert.False(t, out.Players[0].Banned, "with the antibot off nobody is serving")
}
