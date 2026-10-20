package list_place_reports_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_place_reports_handler"
)

type stubQuery struct {
	answer *supportv1.ListPlaceReportsResponse
	err    error

	since    time.Time
	limit    uint32
	openOnly bool
}

func (s *stubQuery) PlaceReports(
	_ context.Context, since time.Time, limit uint32, openOnly bool,
) (*supportv1.ListPlaceReportsResponse, error) {
	s.since, s.limit, s.openOnly = since, limit, openOnly
	return s.answer, s.err
}

func list(t *testing.T, query *stubQuery, req *supportv1.ListPlaceReportsRequest) (*supportv1.ListPlaceReportsResponse, error) {
	t.Helper()

	res, err := list_place_reports_handler.New(query).ListPlaceReports(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("ListPlaceReports failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheWindowReachesTheQueryAndItsAnswerComesBack(t *testing.T) {
	since := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	query := &stubQuery{answer: &supportv1.ListPlaceReportsResponse{Reports: []*supportv1.PlaceReport{{Text: "a swastika"}}}}

	answer, err := list(t, query, &supportv1.ListPlaceReportsRequest{Since: timestamppb.New(since), Limit: 20, OpenOnly: true})
	require.NoError(t, err)

	assert.True(t, since.Equal(query.since))
	assert.Equal(t, uint32(20), query.limit)
	assert.True(t, query.openOnly)
	assert.Same(t, query.answer, answer)
}

func TestNoSinceIsEveryReport(t *testing.T) {
	query := &stubQuery{answer: &supportv1.ListPlaceReportsResponse{}}

	_, err := list(t, query, &supportv1.ListPlaceReportsRequest{})
	require.NoError(t, err)

	assert.True(t, query.since.IsZero())
}

func TestAFailedReadIsLeftToTheErrorNet(t *testing.T) {
	_, err := list(t, &stubQuery{err: errors.New("postgres is down")}, &supportv1.ListPlaceReportsRequest{})

	assert.ErrorContains(t, err, "postgres is down")
}
