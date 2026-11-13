package player_forgotten_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/subscribers/player_forgotten_subscriber"
)

type store struct {
	forgotten []ratings.AccountID
}

func (s *store) ForgetAccount(_ context.Context, account ratings.AccountID) error {
	s.forgotten = append(s.forgotten, account)
	return nil
}

func TestADeletedAccountIsBlankedFromItsQuizAnswers(t *testing.T) {
	kept := &store{}
	const gone = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

	err := player_forgotten_subscriber.New(forget_account_usecase.New(kept)).Handle(t.Context(),
		&playerv1.PlayerForgotten{AccountId: gone})

	require.NoError(t, err)
	account, err := ratings.AccountIDOf(gone)
	require.NoError(t, err)
	assert.Equal(t, []ratings.AccountID{account}, kept.forgotten)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	kept := &store{}

	err := player_forgotten_subscriber.New(forget_account_usecase.New(kept)).Handle(t.Context(),
		&playerv1.PlayerForgotten{})

	require.ErrorIs(t, err, ratings.ErrInvalidAccount)
	assert.Empty(t, kept.forgotten)
}
