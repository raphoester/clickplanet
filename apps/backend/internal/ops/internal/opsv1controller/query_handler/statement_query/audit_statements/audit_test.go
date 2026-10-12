package audit_statements_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler/statement_query/audit_statements"
)

type stubQuery struct {
	rows *opsv1.QueryResponse
	err  error
}

func (s stubQuery) Rows(context.Context, string, uint32) (*opsv1.QueryResponse, error) {
	return s.rows, s.err
}

func run(t *testing.T, inner stubQuery) (string, *opsv1.QueryResponse, error) {
	t.Helper()

	var logs bytes.Buffer
	audited := audit_statements.New(inner, slog.New(slog.NewTextHandler(&logs, nil)))
	rows, err := audited.Rows(access.WithCaller(t.Context(), "claude-cloud.access"), "SELECT count(*) FROM planet.tiles", 10)

	return logs.String(), rows, err
}

func TestAStatementIsLoggedWithItsCallerAndWhatItAnswered(t *testing.T) {
	answer := &opsv1.QueryResponse{Rows: []*structpb.ListValue{{}, {}}, Truncated: true}

	logs, rows, err := run(t, stubQuery{rows: answer})

	require.NoError(t, err)
	assert.Same(t, answer, rows)
	assert.Contains(t, logs, "level=INFO")
	assert.Contains(t, logs, `msg="ops statement"`)
	assert.Contains(t, logs, "caller=claude-cloud.access")
	assert.Contains(t, logs, `statement="SELECT count(*) FROM planet.tiles"`)
	assert.Contains(t, logs, "rows=2")
	assert.Contains(t, logs, "truncated=true")
}

func TestAStatementThatFailedIsLoggedLouderWithWhy(t *testing.T) {
	refused := errors.New("cannot execute DELETE in a read-only transaction")

	logs, _, err := run(t, stubQuery{err: refused})

	require.ErrorIs(t, err, refused)
	assert.Contains(t, logs, "level=WARN")
	assert.Contains(t, logs, `msg="ops statement failed"`)
	assert.Contains(t, logs, "caller=claude-cloud.access")
	assert.Contains(t, logs, "cannot execute DELETE in a read-only transaction")
}
