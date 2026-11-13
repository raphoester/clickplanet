package notices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
)

const (
	ada   = charges.Holder("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	grace = charges.Holder("7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f")
)

func attend(t *testing.T, board *notices.Board, holder charges.Holder) <-chan notices.Event {
	t.Helper()

	events, leave := board.Attend(holder)
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
	gain := charges.Gain{After: charges.Held{Bomb: true, Shields: 12}}

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

	board.PublishGranted(charges.NoHolder, charges.Gain{After: charges.Held{Bomb: true}})

	assert.Empty(t, nobody)
}

func TestAGrantToAnAccountWithNoStreamIsDropped(t *testing.T) {
	assert.NotPanics(t, func() { notices.New().PublishGranted(ada, charges.Gain{After: charges.Held{Bomb: true}}) })
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
	events, leave := board.Attend(charges.NoHolder)
	leave()

	board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 1})

	assert.Empty(t, events)
}
