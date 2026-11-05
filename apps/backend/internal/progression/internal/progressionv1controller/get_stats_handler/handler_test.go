package get_stats_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *progressionv1.GetStatsResponse
	err    error
	asked  []accounts.AccountID
}

func (s *stubQuery) Stats(_ context.Context, account accounts.AccountID) (*progressionv1.GetStatsResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func TestTheCallersAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &progressionv1.GetStatsResponse{Stats: &progressionv1.Stats{TilesTaken: 42, StreakCurrent: 3}}}

	res, err := get_stats_handler.New(query).GetStats(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&progressionv1.GetStatsRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := accounts.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, query.asked)
}

func TestACallerWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := get_stats_handler.New(query).GetStats(t.Context(), connect.NewRequest(&progressionv1.GetStatsRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := get_stats_handler.New(&stubQuery{err: failure}).GetStats(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&progressionv1.GetStatsRequest{}))

	assert.ErrorIs(t, err, failure)
}
