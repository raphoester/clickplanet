package ledger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func TestARetakeByTheSameScopeKeepsTheOwnerFromBeforeItsFirstTake(t *testing.T) {
	first := ledger.Taking{Tile: 7, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start}
	retake := ledger.Taking{Tile: 7, Scope: "1.2.3.4", Country: "ps", Previous: "fr", At: start.Add(time.Second)}

	assert.Equal(t, "de", retake.Over(first, true).Previous)
}

func TestATakeBySomebodyElseKeepsItsOwnPrevious(t *testing.T) {
	first := ledger.Taking{Tile: 7, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start}
	cover := ledger.Taking{Tile: 7, Scope: "5.6.7.8", Country: "de", Previous: "fr", At: start}

	assert.Equal(t, cover, cover.Over(first, true))
	assert.Equal(t, cover, cover.Over(ledger.Taking{}, false))
}

func TestARestorationOnlyAppliesToATileStillWearingThePaint(t *testing.T) {
	taking := ledger.Taking{Tile: 7, Scope: "1.2.3.4", Country: "fr", Previous: "de"}

	assert.True(t, taking.WornBy("fr"))
	assert.False(t, taking.WornBy("de"))
	assert.Equal(t, clicks.Restoration{Tile: 7, From: "fr", To: "de"}, taking.Restoration())
}

func TestPlayersAreGatheredByScopeLatestFirst(t *testing.T) {
	players := ledger.Players([]ledger.Taking{
		{Tile: 1, Scope: "bot", At: start.Add(2 * time.Second)},
		{Tile: 2, Scope: "player", At: start.Add(time.Second)},
		{Tile: 3, Scope: "bot", At: start},
		{Tile: 4, Scope: "also", At: start.Add(time.Second)},
	})

	assert.Equal(t, []ledger.Player{
		{Scope: "bot", Tiles: 2, FirstAt: start, LastAt: start.Add(2 * time.Second)},
		{Scope: "also", Tiles: 1, FirstAt: start.Add(time.Second), LastAt: start.Add(time.Second)},
		{Scope: "player", Tiles: 1, FirstAt: start.Add(time.Second), LastAt: start.Add(time.Second)},
	}, players, "equal times fall back to scope order")
}

func TestTopCutsToTheLimitOrTheDefault(t *testing.T) {
	players := make([]ledger.Player, 30)

	assert.Len(t, ledger.Top(players, 3), 3)
	assert.Len(t, ledger.Top(players, 0), 20)
	assert.Len(t, ledger.Top(players[:2], 5), 2)
}

func TestAServingPlayerCarriesTheSentence(t *testing.T) {
	player := ledger.Player{Scope: "bot"}
	player.Serving(antibot.Sentence{Offence: 2, Until: start})

	assert.Equal(t, ledger.Player{Scope: "bot", Banned: true, BannedUntil: start, Offence: 2}, player)
}

func TestRetentionForgetsTakesPastIt(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)
	takings.Put(ledger.Taking{Tile: 1, Scope: "1.2.3.4", At: start})
	takings.Put(ledger.Taking{Tile: 2, Scope: "1.2.3.4", At: start.Add(30 * time.Minute)})

	clock.Advance(61 * time.Minute)
	ledger.NewRetention(ledger.Config{Retention: time.Hour}, takings, clock).Sweep()

	remaining := takings.TakenBy("1.2.3.4")
	require.Len(t, remaining, 1)
	assert.Equal(t, uint32(2), remaining[0].Tile)
}

type stubTiles struct {
	owners map[uint32]string
	err    error
}

func (s *stubTiles) Owner(tile uint32) (string, bool) { return s.owners[tile], true }

func (s *stubTiles) Set(_ context.Context, tile uint32, value string) error {
	if s.err != nil {
		return s.err
	}
	s.owners[tile] = value
	return nil
}

func (s *stubTiles) SetBoosted(ctx context.Context, tile uint32, value string) error {
	return s.Set(ctx, tile, value)
}

func TestRecordingNotesTheCallersScopeAndOnlyAChange(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{1: "de", 2: "fr"}}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)
	recording := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start))

	ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::1")

	require.NoError(t, recording.Set(ctx, 1, "fr"))
	require.NoError(t, recording.SetBoosted(ctx, 2, "fr"))

	assert.Equal(t, []ledger.Taking{{Tile: 1, Scope: "2001:db8::/64", Country: "fr", Previous: "de", At: start}},
		takings.TakenBy("2001:db8::/64"), "a v6 caller is its /64, and a tile it already held is no take")
}

func TestRecordingARetakeKeepsTheFirstOwner(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{7: "de"}}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)
	recording := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start))

	ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

	require.NoError(t, recording.Set(ctx, 7, "fr"))
	require.NoError(t, recording.Set(ctx, 7, "ps"))

	assert.Equal(t, []ledger.Taking{{Tile: 7, Scope: "1.2.3.4", Country: "ps", Previous: "de", At: start}},
		takings.TakenBy("1.2.3.4"))
}

func TestRecordingNotesNothingForAFailedWrite(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{}, err: errors.New("out of range")}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, nil)

	err := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start)).Set(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), 1, "fr")
	require.ErrorIs(t, err, tiles.err)

	assert.Empty(t, takings.TakenBy("1.2.3.4"))
}
