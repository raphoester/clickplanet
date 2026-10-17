package get_operator_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func TestTheOperatorTheGuardAdmittedIsTheAnswerAndIsNeverCached(t *testing.T) {
	ctx := cpctx.AddOperatorToContext(t.Context(), "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

	res, err := get_operator_handler.New().GetOperator(ctx, connect.NewRequest(&authv1.GetOperatorRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", res.Msg.GetAccountId())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestACallNoGuardAdmittedIsUnauthenticated(t *testing.T) {
	_, err := get_operator_handler.New().GetOperator(t.Context(), connect.NewRequest(&authv1.GetOperatorRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
