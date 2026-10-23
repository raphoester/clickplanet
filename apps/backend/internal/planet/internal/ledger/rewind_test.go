package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func rewindOf(area []uint32, at time.Time, events ...ledger.Event) *ledger.Rewind {
	rewind := ledger.NewRewind(area, at)
	for _, event := range events {
		rewind.See(event)
	}
	return rewind
}

func TestEachTileGoesBackToWhatItHeldBeforeItsFirstChangeSince(t *testing.T) {
	before := start.Add(time.Hour)
	rewind := rewindOf([]uint32{1, 2, 3}, before,
		ledger.Taking{Tile: 1, Country: "it", At: start},
		ledger.Taking{Tile: 1, Country: "de", Previous: "it", At: before},
		ledger.Taking{Tile: 1, Country: "at", Previous: "de", At: before.Add(time.Minute)},
		ledger.Taking{Tile: 2, Country: "de", At: before.Add(time.Minute)},
		ledger.Taking{Tile: 3, Country: "de", Previous: "fr", At: start},
	)

	assert.Equal(t, []clicks.Restoration{
		{Tile: 1, From: "at", To: "it"},
		{Tile: 2, From: "de", To: ""},
	}, rewind.Restorations(owners{1: "at", 2: "de", 3: "de"}), "a tile changed before only is left")
}

func TestATileBackToWhatItHeldOrOutsideTheAreaIsLeft(t *testing.T) {
	before := start.Add(time.Hour)
	rewind := rewindOf([]uint32{1}, before,
		ledger.Taking{Tile: 1, Country: "de", Previous: "fr", At: before},
		ledger.Taking{Tile: 1, Country: "fr", Previous: "de", At: before.Add(time.Minute)},
		ledger.Taking{Tile: 9, Country: "de", Previous: "fr", At: before},
	)

	assert.Empty(t, rewind.Restorations(owners{1: "fr", 9: "de"}))
}

func TestABombIsRewoundToTheFlagsItCleared(t *testing.T) {
	before := start.Add(time.Hour)
	rewind := rewindOf([]uint32{1, 2}, before, ledger.Bombing{At: before, Blast: clicks.Blast{
		Cleared: []uint32{1, 2}, Owners: []string{"fr", "es"},
	}})

	assert.Equal(t, []clicks.Restoration{
		{Tile: 1, From: "", To: "fr"},
		{Tile: 2, From: "de", To: "es"},
	}, rewind.Restorations(owners{2: "de"}))
}

func TestTheLedgerReachesBackItsRetention(t *testing.T) {
	reach := ledger.NewReach(ledger.Config{Retention: time.Hour}, cptime.NewFixedClock(start))

	assert.Equal(t, start.Add(-time.Hour), reach.Since())
}
