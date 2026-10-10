package query_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler/statement_query"
)

type asked struct {
	statement string
	limit     uint32
}

type stubQuery struct {
	rows  *opsv1.QueryResponse
	err   error
	asked *asked
}

func (s stubQuery) Rows(_ context.Context, statement string, limit uint32) (*opsv1.QueryResponse, error) {
	*s.asked = asked{statement: statement, limit: limit}
	return s.rows, s.err
}

func ask(t *testing.T, query stubQuery) (*connect.Response[opsv1.QueryResponse], error) {
	t.Helper()

	query.asked = &asked{}
	//nolint:wrapcheck // the test reads the code off the error as it came.
	return query_handler.New(query).Query(t.Context(), connect.NewRequest(&opsv1.QueryRequest{Statement: "SELECT 1"}))
}

func TestQueryAnswersWhatTheQueryReadsForTheStatementAndTheLimitAsked(t *testing.T) {
	one, err := structpb.NewList([]any{"take", 12})
	require.NoError(t, err)
	rows := &opsv1.QueryResponse{Columns: []string{"kind", "count"}, Rows: []*structpb.ListValue{one}, Truncated: true}
	got := &asked{}

	res, err := query_handler.New(stubQuery{rows: rows, asked: got}).Query(t.Context(), connect.NewRequest(
		&opsv1.QueryRequest{Statement: "SELECT kind, count(*) FROM planet.ledger_events GROUP BY kind", Limit: 50},
	))

	require.NoError(t, err)
	assert.True(t, proto.Equal(rows, res.Msg))
	assert.Equal(t, asked{statement: "SELECT kind, count(*) FROM planet.ledger_events GROUP BY kind", limit: 50}, *got)
}

func TestWhatTheCallerGotWrongIsInvalidArgumentAndSaysWhatPostgresSaid(t *testing.T) {
	for _, refusal := range []error{
		statement_query.ErrNoStatement,
		statement_query.ErrLimitTooHigh,
		fmt.Errorf("%w: cannot execute DELETE in a read-only transaction", statement_query.ErrRefused),
	} {
		_, err := ask(t, stubQuery{err: refusal})

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), refusal)
		require.ErrorContains(t, err, refusal.Error())
	}
}

func TestAStatementThatTookTooLongIsDeadlineExceeded(t *testing.T) {
	_, err := ask(t, stubQuery{err: statement_query.ErrTooSlow})

	assert.Equal(t, connect.CodeDeadlineExceeded, connect.CodeOf(err))
}

func TestPostgresThatCannotBeReadIsUnavailableAndSaysWhy(t *testing.T) {
	_, err := ask(t, stubQuery{
		err: fmt.Errorf("%w: password authentication failed for user \"ops_reader\"", statement_query.ErrUnreachable),
	})

	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.ErrorContains(t, err, "password authentication failed")
}

func TestAFailureNobodyNamedIsLeftToTheErrorNet(t *testing.T) {
	broken := errors.New("failed to scan a row")

	_, err := ask(t, stubQuery{err: broken})

	require.ErrorIs(t, err, broken)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
