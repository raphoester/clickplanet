package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func TestASpanIsCutIntoSlotsThatStartOnTheirStep(t *testing.T) {
	until := start.Add(7 * time.Minute)

	for _, test := range []struct {
		span  time.Duration
		step  time.Duration
		slots int
	}{
		{span: time.Hour, step: 5 * time.Minute, slots: 12},
		{span: 6 * time.Hour, step: 5 * time.Minute, slots: 72},
		{span: 7 * time.Hour, step: 15 * time.Minute, slots: 28},
		{span: 24 * time.Hour, step: 15 * time.Minute, slots: 96},
		{span: 72 * time.Hour, step: time.Hour, slots: 72},
	} {
		activity, err := ledger.NewActivity(until, test.span)
		require.NoError(t, err)

		summary := activity.Summary()
		assert.Equal(t, test.step, summary.Step, test.span)
		require.Len(t, summary.Slots, test.slots, test.span)
		assert.Equal(t, until.Truncate(test.step), summary.Slots[test.slots-1].Start, "the last slot is the one filling now")
		assert.Equal(t, summary.Slots[0].Start, summary.Since)
	}
}

func TestASpanOfNothingOrPastTheRetentionIsRefused(t *testing.T) {
	for _, span := range []time.Duration{0, -time.Hour, 72*time.Hour + time.Second} {
		_, err := ledger.NewActivity(start, span)
		assert.ErrorIs(t, err, ledger.ErrInvalidSpan, span)
	}
}

func TestEachSlotCountsTheClicksTheTilesThatChangedHandsAndWhoTookThem(t *testing.T) {
	until := start.Add(10 * time.Minute)
	activity, err := ledger.NewActivity(until, time.Hour)
	require.NoError(t, err)
	since := activity.Summary().Since
	at := func(minutes int) time.Time { return since.Add(time.Duration(minutes) * time.Minute) }

	for _, event := range []ledger.Event{
		ledger.Taking{Tile: 9, Scope: "A", Country: "fr", At: since.Add(-time.Second)},
		ledger.Taking{Tile: 1, Scope: "A", Country: "fr", At: at(5)},
		ledger.Striking{Tile: 2, Scope: "C", Account: "acc", Country: "fr", Owner: "pl", At: at(6)},
		ledger.Taking{Tile: 3, Scope: "C", Account: "acc", Country: "fr", Previous: "de", At: at(6)},
		ledger.Spreading{Tile: 3, Scope: "D", Account: "acc", Country: "fr", At: at(6), Impacts: []clicks.Impact{
			{Tile: 4, Owner: "de", Outcome: clicks.Taken},
			{Tile: 5, Owner: "pl", Outcome: clicks.Shielded, Shields: 2},
		}},
		ledger.Bombing{Scope: "B", At: at(50), Blast: clicks.Blast{
			Tile: 1, CountryID: "pl", Cleared: []uint32{1, 6}, Owners: []string{"fr", "pl"},
		}},
		ledger.Shielding{Tile: 7, Scope: "B", Country: "pl", Shields: 1, At: at(51)},
		ledger.Taking{Tile: 8, Scope: "A", Country: "fr", At: until},
	} {
		activity.See(event)
	}

	slots := make([]ledger.Slot, 12)
	for i := range slots {
		slots[i].Start = since.Add(time.Duration(i) * 5 * time.Minute)
	}
	slots[1].Clicks, slots[1].Tiles, slots[1].Players = 3, 3, 2
	slots[10].Tiles, slots[10].Players = 2, 1

	assert.Equal(t, ledger.Summary{
		Since: since, Until: until, Step: 5 * time.Minute, Slots: slots, Players: 3,
		Changes: map[uint32]int{1: 2, 3: 1, 4: 1, 6: 1},
		Moves:   map[string]ledger.Moves{"fr": {Won: 3, Lost: 1}, "de": {Lost: 2}, "pl": {Lost: 1}},
	}, activity.Summary(), "a strike is a click that changed no hands, a spread changes hands with no click of its own, "+
		"a bomb clears for nobody, an account is one player on every scope, and nothing outside the span counts")
}
