package box_claimed_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/box_claimed_subscriber"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type paid struct {
	boxes []bonuses.CaughtBox
}

func (p *paid) Execute(_ context.Context, box bonuses.CaughtBox) (pay_caught_box_usecase.Out, error) {
	p.boxes = append(p.boxes, box)
	return pay_caught_box_usecase.Out{Paid: true}, nil
}

func TestABoxCaughtIsPaidUnderItsToken(t *testing.T) {
	useCase := &paid{}

	require.NoError(t, box_claimed_subscriber.New(useCase).Handle(t.Context(), &boxesv1.BoxClaimed{
		AccountId: ada, Token: "a-token", Welcome: true,
	}))

	assert.Equal(t, []bonuses.CaughtBox{{Account: ada, Token: "a-token", Welcome: true}}, useCase.boxes)
}

func TestABoxWithNoAccountOrNoTokenIsRefused(t *testing.T) {
	useCase := &paid{}
	subscriber := box_claimed_subscriber.New(useCase)

	require.ErrorIs(t, subscriber.Handle(t.Context(), &boxesv1.BoxClaimed{AccountId: "nope", Token: "t"}),
		bonuses.ErrInvalidAccount)
	require.Error(t, subscriber.Handle(t.Context(), &boxesv1.BoxClaimed{AccountId: ada}))
	assert.Empty(t, useCase.boxes)
}
