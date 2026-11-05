package signed_in_told_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/tell_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func TestTheAccountSignedInToIsToldWhateverTheBrowserWasOn(t *testing.T) {
	for _, previous := range []string{"", "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60", "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"} {
		outbox := &cpatomic.FakeOutbox{}

		err := signed_in_told_subscriber.New(tell_sign_in_usecase.New(cpatomic.FakeTransactioner{}, outbox)).Handle(
			t.Context(), &authv1.SignedIn{PreviousAccountId: previous, AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"})

		require.NoError(t, err)
		require.Len(t, outbox.Recorded(), 1)
		assert.True(t, proto.Equal(&playerv1.PlayerSignedIn{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"},
			outbox.Recorded()[0]))
	}
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	err := signed_in_told_subscriber.New(tell_sign_in_usecase.New(cpatomic.FakeTransactioner{}, outbox)).Handle(
		t.Context(), &authv1.SignedIn{})

	require.ErrorIs(t, err, players.ErrInvalidAccount)
	assert.Empty(t, outbox.Recorded())
}
