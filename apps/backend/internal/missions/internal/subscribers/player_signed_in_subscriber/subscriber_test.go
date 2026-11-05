package player_signed_in_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/player_signed_in_subscriber"
)

type recordingUseCase struct {
	accounts []missions.AccountID
}

func (u *recordingUseCase) Execute(_ context.Context, account missions.AccountID) error {
	u.accounts = append(u.accounts, account)
	return nil
}

func TestTheAccountThatSignedInIsPaid(t *testing.T) {
	useCase := &recordingUseCase{}
	account, err := missions.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)

	require.NoError(t, player_signed_in_subscriber.New(useCase).Handle(t.Context(),
		&playerv1.PlayerSignedIn{AccountId: account.String()}))

	assert.Equal(t, []missions.AccountID{account}, useCase.accounts)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_signed_in_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &playerv1.PlayerSignedIn{})

	assert.ErrorIs(t, err, missions.ErrInvalidAccount)
}
