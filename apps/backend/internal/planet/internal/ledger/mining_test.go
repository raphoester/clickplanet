package ledger_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var mining = ledger.Mining{Tile: 5, Scope: "A", Account: "guest", Country: "fr", At: start, Impacts: []clicks.Impact{
	{Tile: 4, Outcome: clicks.Taken},
	{Tile: 5, Owner: "de", Outcome: clicks.Taken},
	{Tile: 6, Owner: "it", Outcome: clicks.Shielded, Shields: 2},
}}

func TestAMineGoneOffReplaysAsOneTakePerTileItsBlastTookAndOneStrikePerShieldItStruck(t *testing.T) {
	assert.Equal(t, []ledger.Taking{
		{Tile: 4, Scope: "A", Account: "guest", Country: "fr", At: start},
		{Tile: 5, Scope: "A", Account: "guest", Country: "fr", Previous: "de", At: start},
	}, changes(mining))

	var strikes []ledger.Strike
	mining.Strikes(func(strike ledger.Strike) { strikes = append(strikes, strike) })
	assert.Equal(t, []ledger.Strike{{Tile: 6, Account: "guest", Country: "fr", Owner: "it", At: start}}, strikes)
}

func TestAMineGoneOffIsWrittenDownOnItsTileAndReadBack(t *testing.T) {
	entry, err := mining.Entry()
	require.NoError(t, err)
	assert.Equal(t, "mine", entry.Kind)
	assert.Equal(t, uint32(5), entry.Tile)

	read, err := ledger.EventOf(entry)
	require.NoError(t, err)
	assert.Equal(t, mining, read)
}

type minefield struct {
	detonations []clicks.Detonation
	err         error
}

func (f minefield) Lay(context.Context, clicks.Mine) (clicks.MapVersion, error) {
	return clicks.MapVersion{}, f.err
}

func (f minefield) Detonate(context.Context, int) ([]clicks.Detonation, error) {
	return f.detonations, f.err
}

func detonationOf(tile uint32, scope string, impacts ...clicks.Impact) clicks.Detonation {
	return clicks.Detonation{
		Mine:    clicks.MineSnapshot{Tile: tile, Flag: "fr", Account: carrier, Scope: scope},
		Impacts: impacts,
	}
}

func TestEveryBlastIsRecordedAsItsPlayersActBeforeTheFortifiesAndTheKnockOuts(t *testing.T) {
	events := &recordedEvents{}
	fortified := &clicks.Fortification{Landmass: 3, Country: "fr", Tile: 4, Tiles: 2}
	knocked := &clicks.KnockOut{Country: "de", By: "fr", Tile: 6, Cleared: []uint32{8}}
	recording := ledger.NewMineRecording(minefield{detonations: []clicks.Detonation{
		detonationOf(5, "203.0.113.7", clicks.Impact{Tile: 4, Owner: "de", Outcome: clicks.Taken, Fortified: fortified}),
		detonationOf(7, "203.0.113.7", clicks.Impact{Tile: 6, Owner: "de", Outcome: clicks.Taken, KnockedOut: knocked}),
	}}, events, events, cptime.NewFixedClock(start))

	_, err := recording.Detonate(t.Context(), 2)
	require.NoError(t, err)

	require.Len(t, events.appended, 4)
	assert.Equal(t, ledger.Mining{
		Tile: 5, Scope: "203.0.113.7", Account: carrier.String(), Country: "fr", At: start,
		Impacts: []clicks.Impact{{Tile: 4, Owner: "de", Outcome: clicks.Taken, Fortified: fortified}},
	}, events.appended[0])
	assert.IsType(t, ledger.Mining{}, events.appended[1], "a later blast was written before the fortify")
	assert.IsType(t, ledger.Fortifying{}, events.appended[2])
	assert.IsType(t, ledger.KnockingOut{}, events.appended[3])
}

func TestABlastThatChangedNoTileRecordsNothingAndAFailureStillRecordsWhatWentOff(t *testing.T) {
	events := &recordedEvents{}
	down := errors.New("tile out of range")
	recording := ledger.NewMineRecording(minefield{detonations: []clicks.Detonation{
		detonationOf(5, "203.0.113.7", clicks.Impact{Tile: 5, Owner: "fr", Outcome: clicks.Unchanged}),
		detonationOf(7, "203.0.113.7", clicks.Impact{Tile: 6, Owner: "de", Outcome: clicks.Shielded, Shields: 1}),
	}, err: down}, events, events, cptime.NewFixedClock(start))

	detonations, err := recording.Detonate(t.Context(), 2)

	require.ErrorIs(t, err, down)
	assert.Len(t, detonations, 2)
	require.Len(t, events.appended, 1, "a shield struck is a change")
	assert.Equal(t, uint32(7), events.appended[0].(ledger.Mining).Tile)
}
