package set_shown_in_clips_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_shown_in_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_shown_in_clips_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const (
	named   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	unnamed = "9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a"
)

func handler(t *testing.T) (set_shown_in_clips_handler.SetShownInClipsHandler, *inmemory_player_store.Store) {
	t.Helper()

	store := inmemory_player_store.New()
	account, err := players.AccountIDOf(named)
	require.NoError(t, err)
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(account, "Ada_L", time.Now())))

	return set_shown_in_clips_handler.New(set_shown_in_clips_usecase.New(store)), store
}

func setShown(t *testing.T, h set_shown_in_clips_handler.SetShownInClipsHandler, caller string, shown bool) (bool, error) {
	t.Helper()

	res, err := h.SetShownInClips(cpctx.AddAccountToContext(t.Context(), caller), connect.NewRequest(&playerv1.SetShownInClipsRequest{Shown: shown}))
	if err != nil {
		return false, fmt.Errorf("SetShownInClips failed: %w", err)
	}
	return res.Msg.GetShown(), nil
}

func TestTheChoiceIsKeptAndAnswered(t *testing.T) {
	h, store := handler(t)
	account, err := players.AccountIDOf(named)
	require.NoError(t, err)

	shown, err := setShown(t, h, named, false)
	require.NoError(t, err)
	assert.False(t, shown)
	assert.False(t, store.ShownInClips(account))

	shown, err = setShown(t, h, named, true)
	require.NoError(t, err)
	assert.True(t, shown)
	assert.True(t, store.ShownInClips(account))
}

func TestAnAccountWithNoUsernameIsFailedPrecondition(t *testing.T) {
	h, _ := handler(t)

	_, err := setShown(t, h, unnamed, false)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	h, _ := handler(t)

	_, err := h.SetShownInClips(t.Context(), connect.NewRequest(&playerv1.SetShownInClipsRequest{Shown: false}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
