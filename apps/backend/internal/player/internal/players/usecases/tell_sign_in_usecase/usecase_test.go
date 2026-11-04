package tell_sign_in_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/tell_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func TestEverySignInIsToldToTheOtherModules(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}
	ada := players.AccountID{15: 1}
	useCase := tell_sign_in_usecase.New(cpatomic.FakeTransactioner{}, outbox)

	require.NoError(t, useCase.Execute(t.Context(), ada))
	require.NoError(t, useCase.Execute(t.Context(), ada))

	require.Len(t, outbox.Recorded(), 2, "a second sign-in is told too: the module that hears it decides what it changes")
	for _, told := range outbox.Recorded() {
		assert.True(t, proto.Equal(&playerv1.PlayerSignedIn{AccountId: ada.String()}, told))
	}
}
