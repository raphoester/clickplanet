package count_arrivals_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/count_arrivals_handler"
)

type window struct {
	since time.Time
	until time.Time
}

type stubQuery struct {
	answer *authv1.CountArrivalsResponse
	asked  []window
}

func (s *stubQuery) Counts(_ context.Context, since, until time.Time) (*authv1.CountArrivalsResponse, error) {
	s.asked = append(s.asked, window{since: since, until: until})
	return s.answer, nil
}

func counted() *stubQuery {
	return &stubQuery{answer: &authv1.CountArrivalsResponse{Counts: []*authv1.ArrivalCount{
		{Day: "2026-11-20", Source: authv1.ArrivalSource_ARRIVAL_SOURCE_TIKTOK, Accounts: 4},
	}}}
}

func TestTheCountsOfTheWindowAskedAreAnswered(t *testing.T) {
	query := counted()
	since := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)

	answer, err := count_arrivals_handler.New(query).CountArrivals(t.Context(), connect.NewRequest(&authv1.CountArrivalsRequest{
		Since: timestamppb.New(since), Until: timestamppb.New(until),
	}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, answer.Msg))
	assert.Equal(t, []window{{since: since, until: until}}, query.asked)
}

func TestAnUnsetBoundIsNoBound(t *testing.T) {
	query := counted()

	_, err := count_arrivals_handler.New(query).CountArrivals(t.Context(), connect.NewRequest(&authv1.CountArrivalsRequest{}))

	require.NoError(t, err)
	assert.Equal(t, []window{{}}, query.asked)
}
