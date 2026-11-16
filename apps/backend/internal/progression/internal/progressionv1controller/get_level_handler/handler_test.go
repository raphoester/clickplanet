package get_level_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_level_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *progressionv1.GetLevelResponse
	asked  []accounts.AccountID
}

func (s *stubQuery) Level(_ context.Context, account accounts.AccountID) (*progressionv1.GetLevelResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, nil
}

func TestTheCallersAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &progressionv1.GetLevelResponse{Level: &progressionv1.Level{Level: 3, Xp: 200, LevelXp: 112, NextLevelXp: 378}}}

	res, err := get_level_handler.New(query).GetLevel(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&progressionv1.GetLevelRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := accounts.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, query.asked)
}

func TestACallerWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := get_level_handler.New(query).GetLevel(t.Context(), connect.NewRequest(&progressionv1.GetLevelRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}
