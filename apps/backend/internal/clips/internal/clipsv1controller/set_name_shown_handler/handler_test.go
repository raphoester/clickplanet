package set_name_shown_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_shown_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/set_name_shown_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func TestTheCallerKeepsTheirOwnNameOut(t *testing.T) {
	const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01"
	store := optouts.NewMemoryStore()

	answer, err := set_name_shown_handler.New(set_name_shown_usecase.New(store)).SetNameShown(
		cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&clipsv1.SetNameShownRequest{NameShown: false}))
	require.NoError(t, err)

	assert.False(t, answer.Msg.GetNameShown())
	account, err := casts.AccountIDOf(ada)
	require.NoError(t, err)
	out, err := store.Among(t.Context(), []optouts.AccountID{account})
	require.NoError(t, err)
	assert.Equal(t, []optouts.AccountID{account}, out)
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := set_name_shown_handler.New(set_name_shown_usecase.New(optouts.NewMemoryStore())).SetNameShown(
		t.Context(), connect.NewRequest(&clipsv1.SetNameShownRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
}
