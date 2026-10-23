package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func TestAnExcerptIsEveryChangeTheStreamShowedWithWhoMadeIt(t *testing.T) {
	excerpt := ledger.NewExcerpt(start, start.Add(time.Hour))
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }

	for _, event := range []ledger.Event{
		ledger.Taking{Tile: 9, Scope: "A", Country: "fr", At: start.Add(-time.Second)},
		ledger.Taking{Tile: 1, Scope: "A", Country: "fr", At: at(1)},
		ledger.Striking{Tile: 2, Scope: "C", Account: "acc", Country: "fr", Owner: "pl", Shields: 3, At: at(2)},
		ledger.Spreading{Tile: 3, Scope: "D", Account: "acc", Country: "fr", At: at(3), Impacts: []clicks.Impact{
			{Tile: 4, Owner: "de", Outcome: clicks.Taken},
		}},
		ledger.Bombing{Scope: "B", At: at(4), Blast: clicks.Blast{
			Tile: 1, CountryID: "pl", Cleared: []uint32{1}, Owners: []string{"fr"}, Struck: []uint32{5}, Left: []int{2},
		}},
		ledger.Shielding{Tile: 7, Scope: "B", Country: "pl", Shields: 1, At: at(5)},
		ledger.Taking{Tile: 8, Scope: "A", Country: "fr", At: start.Add(time.Hour)},
	} {
		excerpt.See(event)
	}

	assert.Equal(t, []ledger.Line{
		{At: at(1), By: ledger.Caller{Scope: "A"}, Update: clicks.TileUpdate{Tile: 1, Value: "fr", Clicked: true}},
		{
			At: at(2), By: ledger.Caller{Account: "acc"},
			Update: clicks.TileUpdate{Tile: 2, Value: "pl", Previous: "pl", Clicked: true, Shields: 3},
		},
		{At: at(3), By: ledger.Caller{Account: "acc"}, Update: clicks.TileUpdate{Tile: 4, Value: "fr", Previous: "de"}},
		{At: at(4), By: ledger.Caller{Scope: "B"}, Update: clicks.TileUpdate{Tile: 1, Previous: "fr"}},
		{At: at(5), By: ledger.Caller{Scope: "B"}, Update: clicks.TileUpdate{Tile: 7, Value: "pl", Previous: "pl", Shields: 1}},
	}, excerpt.Lines(), "a strike is a click that kept the flag, a spread's tile is no click, a bomb empties its tiles, "+
		"an account is one caller on every scope, and nothing outside the window shows")
}
