package get_ledger_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler/changes_query"
)

var at = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

type stubQuery struct {
	since, until time.Time
	page         string
	answer       *planetv1.GetLedgerResponse
	err          error
}

func (s *stubQuery) Ledger(_ context.Context, since, until time.Time, page string) (*planetv1.GetLedgerResponse, error) {
	s.since, s.until, s.page = since, until, page
	return s.answer, s.err
}

func TestTheWindowReachesTheQueryAndItsAnswerTheCaller(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetLedgerResponse{Changes: []*planetv1.LedgerChange{{By: "scope:203.0.113.7"}}}}

	res, err := get_ledger_handler.New(query).GetLedger(t.Context(), connect.NewRequest(&planetv1.GetLedgerRequest{
		Since: timestamppb.New(at), Page: "20000",
	}))
	require.NoError(t, err)

	assert.Equal(t, at, query.since)
	assert.Equal(t, "20000", query.page)
	assert.True(t, query.until.IsZero(), "an unset until is the query's to fill")
	assert.True(t, proto.Equal(query.answer, res.Msg))
}

func TestAnInvalidWindowOrPageIsTheCallersMistake(t *testing.T) {
	for _, mistake := range []error{changes_query.ErrInvalidWindow, changes_query.ErrInvalidPage} {
		query := &stubQuery{err: fmt.Errorf("wrapped: %w", mistake)}

		_, err := get_ledger_handler.New(query).GetLedger(t.Context(), connect.NewRequest(&planetv1.GetLedgerRequest{}))

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), mistake)
	}
}
