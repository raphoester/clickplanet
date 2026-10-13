package list_reports_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler"
)

type stubQuery struct {
	answer *supportv1.ListReportsResponse
	err    error

	since time.Time
	limit uint32
}

func (s *stubQuery) Reports(_ context.Context, since time.Time, limit uint32) (*supportv1.ListReportsResponse, error) {
	s.since, s.limit = since, limit
	return s.answer, s.err
}

func list(t *testing.T, query *stubQuery, req *supportv1.ListReportsRequest) (*supportv1.ListReportsResponse, error) {
	t.Helper()

	res, err := list_reports_handler.New(query).ListReports(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("ListReports failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheWindowReachesTheQueryAndItsAnswerComesBack(t *testing.T) {
	since := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	query := &stubQuery{answer: &supportv1.ListReportsResponse{Reports: []*supportv1.Report{{Text: "it never sleeps"}}}}

	answer, err := list(t, query, &supportv1.ListReportsRequest{Since: timestamppb.New(since), Limit: 20})
	require.NoError(t, err)

	assert.True(t, since.Equal(query.since))
	assert.Equal(t, uint32(20), query.limit)
	assert.Same(t, query.answer, answer)
}

func TestNoSinceIsEveryReport(t *testing.T) {
	query := &stubQuery{answer: &supportv1.ListReportsResponse{}}

	_, err := list(t, query, &supportv1.ListReportsRequest{})
	require.NoError(t, err)

	assert.True(t, query.since.IsZero())
}

func TestAFailedReadIsLeftToTheErrorNet(t *testing.T) {
	_, err := list(t, &stubQuery{err: errors.New("postgres is down")}, &supportv1.ListReportsRequest{})

	assert.ErrorContains(t, err, "postgres is down")
}
