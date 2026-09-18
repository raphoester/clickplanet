package signed_in_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_subscriber"
)

const (
	guest = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	ada   = "7c1e2d3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
)

type move struct{ from, to players.AccountID }

// recordingMoves plays the use case and keeps what it was asked.
type recordingMoves struct {
	moves []move
}

func (r *recordingMoves) Execute(_ context.Context, from, to players.AccountID) error {
	r.moves = append(r.moves, move{from: from, to: to})
	return nil
}

func TestTheVisitMovesFromThePreviousAccount(t *testing.T) {
	moves := &recordingMoves{}

	err := signed_in_subscriber.New(moves).Handle(t.Context(), &authv1.SignedIn{PreviousAccountId: guest, AccountId: ada})

	require.NoError(t, err)
	from, err := players.AccountIDOf(guest)
	require.NoError(t, err)
	to, err := players.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []move{{from: from, to: to}}, moves.moves)
}

func TestABrowserWithNoPreviousAccountMovesNothing(t *testing.T) {
	moves := &recordingMoves{}

	err := signed_in_subscriber.New(moves).Handle(t.Context(), &authv1.SignedIn{AccountId: ada})

	require.NoError(t, err)
	assert.Empty(t, moves.moves)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	moves := &recordingMoves{}

	err := signed_in_subscriber.New(moves).Handle(t.Context(), &authv1.SignedIn{PreviousAccountId: guest})

	require.ErrorIs(t, err, players.ErrInvalidAccount)
	assert.Empty(t, moves.moves)
}
