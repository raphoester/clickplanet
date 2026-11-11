package notices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
)

func attend(t *testing.T, board *notices.Board, payer clicks.Payer) <-chan notices.Event {
	t.Helper()

	events, leave := board.Attend(clicks.CallerOf(payer))
	t.Cleanup(leave)

	return events
}

func TestASpreadClickIsAnnouncedToEveryone(t *testing.T) {
	board := notices.New()
	watchers := []<-chan notices.Event{
		attend(t, board, clicks.Payer{Scope: "scope-a"}),
		attend(t, board, clicks.Payer{Scope: "scope-b", Account: "acc-1", Linked: true}),
	}

	board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 100, Neighbours: []uint32{99, 101}})

	for _, events := range watchers {
		require.Len(t, events, 1)

		spread := <-events
		require.NotNil(t, spread.Spread)
		assert.Equal(t, notices.Spread{CountryID: "fr", Tile: 100, Neighbours: []uint32{99, 101}}, *spread.Spread)
	}
}

func TestAGrantReachesOnlyTheSignedInAccountsOwnStreams(t *testing.T) {
	board := notices.New()
	mine := attend(t, board, clicks.Payer{Scope: "scope-a", Account: "acc-1", Linked: true})
	elsewhere := attend(t, board, clicks.Payer{Scope: "scope-c", Account: "acc-1", Linked: true})
	sameAddress := attend(t, board, clicks.Payer{Scope: "scope-a", Account: "acc-1"})
	other := attend(t, board, clicks.Payer{Scope: "scope-b", Account: "acc-2", Linked: true})

	board.PublishGranted("acc-1", charges.Held{Bomb: true, Shields: 12})

	for _, events := range []<-chan notices.Event{mine, elsewhere} {
		granted := (<-events).Granted
		require.NotNil(t, granted)
		assert.Equal(t, charges.Held{Bomb: true, Shields: 12}, *granted)
	}
	assert.Empty(t, sameAddress, "a guest's stream is its address's, which others share")
	assert.Empty(t, other)
}

func TestAGrantToAnAccountWithNoStreamIsDropped(t *testing.T) {
	assert.NotPanics(t, func() { notices.New().PublishGranted("acc-1", charges.Held{Bomb: true}) })
}

func TestAStreamThatIsNotReadingIsDroppedRatherThanBlocking(t *testing.T) {
	board := notices.New()
	events := attend(t, board, clicks.Payer{Scope: "scope-a"})

	for range 100 {
		board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 1})
	}

	assert.Len(t, events, 32)
}

func TestAStreamLeftHearsNothingMore(t *testing.T) {
	board := notices.New()
	events, leave := board.Attend(clicks.CallerOf(clicks.Payer{Scope: "scope-a"}))
	leave()

	board.PublishSpread(notices.Spread{CountryID: "fr", Tile: 1})

	assert.Empty(t, events)
}
