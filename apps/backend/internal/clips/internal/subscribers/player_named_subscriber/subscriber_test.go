package player_named_subscriber_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/subscribers/player_named_subscriber"
)

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01"

type judging struct {
	err       error
	accounts  []casts.AccountID
	deadlines []time.Duration
}

func (j *judging) Execute(ctx context.Context, account casts.AccountID) error {
	j.accounts = append(j.accounts, account)
	if deadline, set := ctx.Deadline(); set {
		j.deadlines = append(j.deadlines, time.Until(deadline))
	}
	return j.err
}

func TestTheNamedAccountIsJudgedWithinAMinute(t *testing.T) {
	useCase := &judging{}

	require.NoError(t, player_named_subscriber.New(useCase).Handle(t.Context(), &playerv1.PlayerNamed{AccountId: adaID}))

	ada, err := casts.AccountIDOf(adaID)
	require.NoError(t, err)
	assert.Equal(t, []casts.AccountID{ada}, useCase.accounts)
	require.Len(t, useCase.deadlines, 1)
	assert.InDelta(t, player_named_subscriber.Timeout, useCase.deadlines[0], float64(time.Second))
}

func TestAJudgementThatFailsFailsTheEventSoItIsTriedAgain(t *testing.T) {
	failure := errors.New("overloaded")

	err := player_named_subscriber.New(&judging{err: failure}).Handle(t.Context(), &playerv1.PlayerNamed{AccountId: adaID})

	assert.ErrorIs(t, err, failure)
}

func TestAnEventWithNoAccountJudgesNothing(t *testing.T) {
	useCase := &judging{}

	err := player_named_subscriber.New(useCase).Handle(t.Context(), &playerv1.PlayerNamed{AccountId: "Ada"})

	require.ErrorIs(t, err, casts.ErrInvalidAccount)
	assert.Empty(t, useCase.accounts)
}
