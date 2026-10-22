package audit_approve_connection_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/approve_connection_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/approve_connection_usecase/audit_approve_connection"
)

type stub struct {
	out approve_connection_usecase.Out
	err error
}

func (s stub) Execute(context.Context, approve_connection_usecase.In) (approve_connection_usecase.Out, error) {
	return s.out, s.err
}

func TestEveryAppLetInIsLoggedWithWhoLetItInAndWhereItIs(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	in := approve_connection_usecase.In{Request: "the sealed request", Operator: accounts.AccountID{15: 1}}

	_, err := audit_approve_connection.New(stub{out: approve_connection_usecase.Out{
		RedirectURL: "https://claude.ai/api/mcp/auth_callback?code=the-code", Client: "Claude", Host: "claude.ai",
	}}, logger).Execute(t.Context(), in)
	require.NoError(t, err)
	_, err = audit_approve_connection.New(stub{err: errors.New("lapsed")}, logger).Execute(t.Context(), in)
	require.Error(t, err)

	assert.Contains(t, logs.String(),
		`level=WARN msg="admin connection approved" operator=00000000-0000-0000-0000-000000000001 client=Claude host=claude.ai`)
	assert.Contains(t, logs.String(), `msg="admin connection refused"`)
	assert.NotContains(t, logs.String(), "the-code", "the code is the app's")
	assert.NotContains(t, logs.String(), "the sealed request")
}
