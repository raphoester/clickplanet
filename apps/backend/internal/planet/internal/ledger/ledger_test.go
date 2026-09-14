package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

func TestARetakeByTheSameScopeKeepsTheOwnerFromBeforeItsFirstTake(t *testing.T) {
	ledger := New(Config{}, cptime.NewFixedClock(start))

	ledger.Record(7, "1.2.3.4", "de", "fr")
	ledger.Record(7, "1.2.3.4", "fr", "ps")

	assert.Equal(t, []Taking{{Tile: 7, Scope: "1.2.3.4", Country: "ps", Previous: "de", At: start}},
		ledger.TakenBy("1.2.3.4"))
}

func TestATakeBySomebodyElseCoversTheTile(t *testing.T) {
	ledger := New(Config{}, cptime.NewFixedClock(start))

	ledger.Record(7, "1.2.3.4", "de", "fr")
	ledger.Record(7, "5.6.7.8", "fr", "de")

	assert.Empty(t, ledger.TakenBy("1.2.3.4"))
	assert.Len(t, ledger.TakenBy("5.6.7.8"), 1)
	assert.Empty(t, ledger.Painted("fr"))
	assert.Len(t, ledger.Painted("de"), 1)
}

func TestForgetLeavesATileTakenAgainSince(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	ledger := New(Config{}, clock)

	ledger.Record(1, "1.2.3.4", "", "fr")
	ledger.Record(2, "1.2.3.4", "", "fr")
	taken := ledger.TakenBy("1.2.3.4")

	clock.Advance(time.Second)
	ledger.Record(2, "1.2.3.4", "fr", "ps")

	ledger.Forget(taken)

	remaining := ledger.TakenBy("1.2.3.4")
	require.Len(t, remaining, 1)
	assert.Equal(t, uint32(2), remaining[0].Tile)
}

func TestTheSweepForgetsTakesPastRetention(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	ledger := New(Config{Retention: time.Hour}, clock)

	ledger.Record(1, "1.2.3.4", "", "fr")
	clock.Advance(30 * time.Minute)
	ledger.Record(2, "1.2.3.4", "", "fr")
	clock.Advance(31 * time.Minute)

	ledger.sweep()

	remaining := ledger.TakenBy("1.2.3.4")
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
	ledger := New(Config{}, cptime.NewFixedClock(start))
	recording := Recording{Tiles: tiles, Ledger: ledger}

	ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::1")

	require.NoError(t, recording.Set(ctx, 1, "fr"))
	require.NoError(t, recording.SetBoosted(ctx, 2, "fr"))

	assert.Equal(t, []Taking{{Tile: 1, Scope: "2001:db8::/64", Country: "fr", Previous: "de", At: start}},
		ledger.TakenBy("2001:db8::/64"), "a v6 caller is its /64, and a tile it already held is no take")
}

func TestRecordingNotesNothingForAFailedWrite(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{}, err: errors.New("out of range")}
	ledger := New(Config{}, cptime.NewFixedClock(start))

	err := Recording{Tiles: tiles, Ledger: ledger}.Set(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), 1, "fr")
	require.ErrorIs(t, err, tiles.err)

	assert.Empty(t, ledger.TakenBy("1.2.3.4"))
}
