package set_color_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_color_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_color_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const (
	named   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	unnamed = "9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a"
)

func handler(t *testing.T) (set_color_handler.SetColorHandler, *inmemory_player_store.Store) {
	t.Helper()

	store := inmemory_player_store.New()
	account, err := players.AccountIDOf(named)
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: account, Name: "Ada_L", UpdatedAt: time.Now()}))

	return set_color_handler.New(set_color_usecase.New(store)), store
}

func setColor(t *testing.T, h set_color_handler.SetColorHandler, caller string, color playerv1.NameColor) (playerv1.NameColor, error) {
	t.Helper()

	res, err := h.SetColor(cpctx.AddAccountToContext(t.Context(), caller), connect.NewRequest(&playerv1.SetColorRequest{Color: color}))
	if err != nil {
		return 0, fmt.Errorf("SetColor failed: %w", err)
	}
	return res.Msg.GetColor(), nil
}

func TestTheColorIsKeptAndAnswered(t *testing.T) {
	h, store := handler(t)

	color, err := setColor(t, h, named, playerv1.NameColor_NAME_COLOR_VIOLET)

	require.NoError(t, err)
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_VIOLET, color)
	account, err := players.AccountIDOf(named)
	require.NoError(t, err)
	profile, err := store.Profile(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, players.Color(playerv1.NameColor_NAME_COLOR_VIOLET), profile.Color)
}

func TestNoColorGoesBackToTheOneTheNameGives(t *testing.T) {
	h, _ := handler(t)
	_, err := setColor(t, h, named, playerv1.NameColor_NAME_COLOR_VIOLET)
	require.NoError(t, err)

	color, err := setColor(t, h, named, playerv1.NameColor_NAME_COLOR_UNSPECIFIED)

	require.NoError(t, err)
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_UNSPECIFIED, color)
}

func TestAColorTheProtoDoesNotNameIsInvalidArgument(t *testing.T) {
	h, _ := handler(t)

	_, err := setColor(t, h, named, playerv1.NameColor(99))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAnAccountWithNoUsernameIsFailedPrecondition(t *testing.T) {
	h, _ := handler(t)

	_, err := setColor(t, h, unnamed, playerv1.NameColor_NAME_COLOR_VIOLET)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	h, _ := handler(t)

	_, err := h.SetColor(t.Context(), connect.NewRequest(&playerv1.SetColorRequest{Color: playerv1.NameColor_NAME_COLOR_RED}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
