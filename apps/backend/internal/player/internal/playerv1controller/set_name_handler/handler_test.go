package set_name_handler_test

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	account      = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	otherAccount = "5c3a1b2d-8e9f-4a0b-9c1d-2e3f4a5b6c7d"
	guestAccount = "9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a"
)

func handler(t *testing.T) set_name_handler.SetNameHandler {
	t.Helper()

	accounts := set_name_usecase.NewFakeAccounts()
	for _, id := range []string{account, otherAccount} {
		linked, err := players.AccountIDOf(id)
		require.NoError(t, err)
		accounts.Link(linked)
	}

	return set_name_handler.New(set_name_usecase.New(inmemory_player_store.New(), accounts,
		cptime.NewFixedClock(time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC))))
}

func setName(t *testing.T, h set_name_handler.SetNameHandler, caller string, name string) (*playerv1.Profile, error) {
	t.Helper()

	res, err := h.SetName(cpctx.AddAccountToContext(t.Context(), caller), connect.NewRequest(&playerv1.SetNameRequest{Name: name}))
	if err != nil {
		return nil, fmt.Errorf("SetName failed: %w", err)
	}
	return res.Msg.GetProfile(), nil
}

func TestTheNameIsAnsweredAsTyped(t *testing.T) {
	profile, err := setName(t, handler(t), account, "Ada_L")

	require.NoError(t, err)
	assert.Equal(t, account, profile.GetAccountId())
	assert.Equal(t, "Ada_L", profile.GetName())
}

func TestAnInvalidNameIsInvalidArgument(t *testing.T) {
	_, err := setName(t, handler(t), account, "Ada!")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestANameAnotherPlayerHoldsIsAlreadyExists(t *testing.T) {
	h := handler(t)
	_, err := setName(t, h, account, "Ada")
	require.NoError(t, err)

	_, err = setName(t, h, otherAccount, "ada")

	assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestAGuestIsPermissionDenied(t *testing.T) {
	_, err := setName(t, handler(t), guestAccount, "Ada")

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	_, err := handler(t).SetName(t.Context(), connect.NewRequest(&playerv1.SetNameRequest{Name: "Ada"}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
