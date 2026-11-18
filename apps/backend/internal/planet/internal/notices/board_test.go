package notices_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
)

const (
	ada   = charges.Holder("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	grace = charges.Holder("7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f")
)

func attend(t *testing.T, board *notices.Board, holder charges.Holder) <-chan notices.Event {
	t.Helper()

	events, leave := board.Attend(holder, "")
	t.Cleanup(leave)

	return events
}

func TestASpreadClickIsAnnouncedToEveryone(t *testing.T) {
	board := notices.New()
	watchers := []<-chan notices.Event{attend(t, board, charges.NoHolder), attend(t, board, ada)}

	board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 100, Neighbours: []uint32{99, 101}})

	for _, events := range watchers {
		require.Len(t, events, 1)

		spread := <-events
		require.NotNil(t, spread.Spread)
		assert.Equal(t, notices.Spread{CountryID: "fr", Tile: 100, Neighbours: []uint32{99, 101}}, *spread.Spread)
	}
}

func TestAGrantReachesEveryStreamOfTheAccountGrantedAndNoOther(t *testing.T) {
	board := notices.New()
	mine, elsewhere := attend(t, board, ada), attend(t, board, ada)
	other, nobody := attend(t, board, grace), attend(t, board, charges.NoHolder)
	gain := charges.Gain{After: charges.Held{Bombs: 1, Shields: 12}}

	board.PublishGranted(ada, gain)

	for _, events := range []<-chan notices.Event{mine, elsewhere} {
		granted := (<-events).Granted
		require.NotNil(t, granted)
		assert.Equal(t, gain, *granted)
	}
	assert.Empty(t, other, "a guest's token names its own account, which a guest behind its address does not share")
	assert.Empty(t, nobody)
}

func TestAGrantToNobodyReachesNoStream(t *testing.T) {
	board := notices.New()
	nobody := attend(t, board, charges.NoHolder)

	board.PublishGranted(charges.NoHolder, charges.Gain{After: charges.Held{Bombs: 1}})

	assert.Empty(t, nobody)
}

func TestAStreamThatIsNotReadingIsDroppedRatherThanBlocking(t *testing.T) {
	board := notices.New()
	events := attend(t, board, charges.NoHolder)

	for range 100 {
		board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 1})
	}

	assert.Len(t, events, 32)
}

func TestAStreamLeftHearsNothingMore(t *testing.T) {
	board := notices.New()
	events, leave := board.Attend(charges.NoHolder, "")
	leave()

	board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 1})

	assert.Empty(t, events)
}

func attendFor(t *testing.T, board *notices.Board, holder charges.Holder, flag string) <-chan notices.Event {
	t.Helper()

	events, leave := board.Attend(holder, flag)
	t.Cleanup(leave)

	return events
}

func TestAMineLaidReachesTheStreamsOfItsFlagAlone(t *testing.T) {
	board := notices.New()
	french, german, none := attendFor(t, board, ada, "fr"), attendFor(t, board, grace, "de"), attend(t, board, charges.NoHolder)
	laid := notices.MineLaid{Flag: "fr", Tile: 9, Version: clicks.MapVersion{Epoch: 7, Seq: 3}}

	board.PublishMineLaid(laid)

	require.Len(t, french, 1)
	got := (<-french).MineLaid
	require.NotNil(t, got)
	assert.Equal(t, laid, *got)
	assert.Empty(t, german, "another flag never learns of it")
	assert.Empty(t, none)
}

func TestAPledgeMovesEveryStreamOfTheAccountToItsNewFlag(t *testing.T) {
	board := notices.New()
	tab, other := attendFor(t, board, ada, "de"), attendFor(t, board, grace, "de")

	board.Pledged(ada, "fr")
	board.PublishMineLaid(notices.MineLaid{Flag: "de", Tile: 3})
	board.PublishMineLaid(notices.MineLaid{Flag: "fr", Tile: 4})

	require.Len(t, tab, 1)
	assert.Equal(t, uint32(4), (<-tab).MineLaid.Tile, "it hears its new flag's mines, and no more its old flag's")
	require.Len(t, other, 1)
	assert.Equal(t, uint32(3), (<-other).MineLaid.Tile)
}

func TestARallyReachesEveryStreamWhateverItsFlag(t *testing.T) {
	board := notices.New()
	theFlags, another := board.Attend(ada, "fr")
	t.Cleanup(another)
	enemies, leave := board.Attend(grace, "de")
	t.Cleanup(leave)
	nobody := attend(t, board, charges.NoHolder)
	called := notices.RallyCalled{Rally: rallies.Snapshot{Flag: "fr", Centre: clicks.Vec3{Z: 1}, Radius: 0.072}, Left: time.Minute}

	board.PublishRally(called)

	for _, events := range []<-chan notices.Event{theFlags, enemies, nobody} {
		rally := (<-events).Rally
		require.NotNil(t, rally)
		assert.Equal(t, called, *rally)
	}
}
