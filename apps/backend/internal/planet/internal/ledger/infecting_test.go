package ledger_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var infecting = ledger.Infecting{Tile: 5, Scope: "A", Account: "guest", Country: "fr", At: start, Impacts: []clicks.Impact{
	{Tile: 4, Owner: "de", Outcome: clicks.Taken},
	{Tile: 6, Owner: "de", Outcome: clicks.Shielded, Shields: 1},
}}

func TestAVirusReplaysAsOneTakePerTileItTookAndStrikesItsShields(t *testing.T) {
	assert.Equal(t, []ledger.Taking{{Tile: 4, Scope: "A", Account: "guest", Country: "fr", Previous: "de", At: start}},
		changes(infecting))

	var strikes []ledger.Strike
	infecting.Strikes(func(strike ledger.Strike) { strikes = append(strikes, strike) })
	assert.Equal(t, []ledger.Strike{{Tile: 6, Account: "guest", Country: "fr", Owner: "de", At: start}}, strikes)
}

func TestAVirusIsWrittenDownOnTheTileItWasUsedOnAndReadBack(t *testing.T) {
	entry, err := infecting.Entry()
	require.NoError(t, err)
	assert.Equal(t, "virus", entry.Kind)
	assert.Equal(t, uint32(5), entry.Tile)

	read, err := ledger.EventOf(entry)
	require.NoError(t, err)
	assert.Equal(t, infecting, read)
}

type contagion struct {
	virus   clicks.Virus
	impacts []clicks.Impact
}

func (c contagion) Start(_ context.Context, _ clicks.Virus, _ int, _ time.Time) (clicks.Virus, []clicks.Impact, error) {
	return c.virus, c.impacts, nil
}

func (c contagion) Spread(_ context.Context, _ clicks.VirusID, _ int, _ time.Time) (clicks.Virus, []clicks.Impact, error) {
	return c.virus, c.impacts, nil
}

var carrier = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}

func spreadVirus() clicks.Virus {
	return clicks.VirusSnapshot{ID: "v-1", Account: carrier, Scope: "203.0.113.7", Flag: "fr", Victim: "de", Tile: 5}.Restore()
}

func TestAVirusTickIsRecordedAsItsPlayersActWhoeverAsks(t *testing.T) {
	events := &recordedEvents{}
	fortified := &clicks.Fortification{Province: 3, Country: "fr", Tile: 4, Tiles: 2}
	recording := ledger.NewVirusRecording(contagion{virus: spreadVirus(), impacts: []clicks.Impact{
		{Tile: 4, Owner: "de", Outcome: clicks.Taken, Fortified: fortified},
	}}, events, events, cptime.NewFixedClock(start))

	_, err := recording.Spread(t.Context(), "v-1", 32, start)
	require.NoError(t, err)

	require.Len(t, events.appended, 2)
	assert.Equal(t, ledger.Infecting{
		Tile: 5, Scope: "203.0.113.7", Account: carrier.String(), Country: "fr", At: start,
		Impacts: []clicks.Impact{{Tile: 4, Owner: "de", Outcome: clicks.Taken, Fortified: fortified}},
	}, events.appended[0])
	assert.IsType(t, ledger.Fortifying{}, events.appended[1])
	assert.Equal(t, []clicks.Stage{clicks.WorldStage, clicks.WorldStage}, events.stages, "each on the stage the tick ran on")
}

func TestATickThatChangedNoTileRecordsNothing(t *testing.T) {
	events := &recordedEvents{}
	recording := ledger.NewVirusRecording(contagion{virus: spreadVirus()}, events, events, cptime.NewFixedClock(start))

	_, err := recording.Spread(t.Context(), "v-1", 32, start)
	require.NoError(t, err)

	assert.Empty(t, events.appended)
}

type recordedEvents struct {
	appended []ledger.Event
	stages   []clicks.Stage
}

func (r *recordedEvents) Append(on clicks.Stage, event ledger.Event) {
	r.stages = append(r.stages, on)
	r.appended = append(r.appended, event)
}

func (r *recordedEvents) Replay(func(clicks.Stage, ledger.Event)) ledger.Position { return 0 }

// During runs a write on the world's map, on earth-1.
func (r *recordedEvents) During(write func(on clicks.Stage)) { write(clicks.WorldStage) }

func (r *recordedEvents) Forget(ledger.Caller, ledger.Position) {}

func (r *recordedEvents) ForgetBefore(time.Time) {}
