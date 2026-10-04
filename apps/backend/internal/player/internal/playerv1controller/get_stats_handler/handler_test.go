package get_stats_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *playerv1.GetStatsResponse
	err    error
	asked  []players.AccountID
}

func (s *stubQuery) Stats(_ context.Context, account players.AccountID) (*playerv1.GetStatsResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func TestTheCallersAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetStatsResponse{Stats: &playerv1.Stats{TilesTaken: 42, StreakCurrent: 3}}}

	res, err := get_stats_handler.New(query).GetStats(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&playerv1.GetStatsRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := players.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []players.AccountID{account}, query.asked)
}

func TestACallerWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := get_stats_handler.New(query).GetStats(t.Context(), connect.NewRequest(&playerv1.GetStatsRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := get_stats_handler.New(&stubQuery{err: failure}).GetStats(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&playerv1.GetStatsRequest{}))

	assert.ErrorIs(t, err, failure)
}
