package leave_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/leave_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type recordingUseCase struct {
	accounts []players.AccountID
}

func (r *recordingUseCase) Execute(_ context.Context, account players.AccountID) error {
	r.accounts = append(r.accounts, account)
	return nil
}

func TestTheCallerLeaves(t *testing.T) {
	useCase := &recordingUseCase{}
	ctx := cpctx.AddAccountToContext(t.Context(), "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

	_, err := leave_handler.New(useCase).Leave(ctx, connect.NewRequest(&playerv1.LeaveRequest{}))

	require.NoError(t, err)
	require.Len(t, useCase.accounts, 1)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", useCase.accounts[0].String())
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	useCase := &recordingUseCase{}

	_, err := leave_handler.New(useCase).Leave(t.Context(), connect.NewRequest(&playerv1.LeaveRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, useCase.accounts)
}
