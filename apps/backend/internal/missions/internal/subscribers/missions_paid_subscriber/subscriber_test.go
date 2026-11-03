package missions_paid_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/missions_paid_subscriber"
)

type recordingStreams struct {
	accounts []missions.AccountID
}

func (s *recordingStreams) Publish(account missions.AccountID) {
	s.accounts = append(s.accounts, account)
}

func TestTheStreamsOfTheAccountPaidAreTold(t *testing.T) {
	streams := &recordingStreams{}
	account, err := missions.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)

	require.NoError(t, missions_paid_subscriber.New(streams).Handle(t.Context(),
		&missionsv1.MissionsPaid{AccountId: account.String(), PaidInAll: 3}))

	assert.Equal(t, []missions.AccountID{account}, streams.accounts)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	streams := &recordingStreams{}

	err := missions_paid_subscriber.New(streams).Handle(t.Context(), &missionsv1.MissionsPaid{})

	require.ErrorIs(t, err, missions.ErrInvalidAccount)
	assert.Empty(t, streams.accounts)
}
