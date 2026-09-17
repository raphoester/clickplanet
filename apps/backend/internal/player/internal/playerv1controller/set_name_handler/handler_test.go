package set_name_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/set_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func handler() set_name_handler.SetNameHandler {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	return set_name_handler.New(set_name_usecase.New(storage, cptime.NewFixedClock(time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC))))
}

func TestTheCleanedNameIsAnswered(t *testing.T) {
	res, err := handler().SetName(cpctx.AddAccountToContext(t.Context(), account),
		connect.NewRequest(&playerv1.SetNameRequest{Name: " Ada\n"}))

	require.NoError(t, err)
	assert.Equal(t, account, res.Msg.GetProfile().GetAccountId())
	assert.Equal(t, "Ada", res.Msg.GetProfile().GetName())
}

func TestAnInvalidNameIsInvalidArgument(t *testing.T) {
	_, err := handler().SetName(cpctx.AddAccountToContext(t.Context(), account),
		connect.NewRequest(&playerv1.SetNameRequest{Name: "\n"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	_, err := handler().SetName(t.Context(), connect.NewRequest(&playerv1.SetNameRequest{Name: "Ada"}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
