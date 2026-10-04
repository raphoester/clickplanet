package get_history_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubQuery struct {
	history *chatv1.GetHistoryResponse
	err     error
	viewer  *messages.AccountID
}

func (s stubQuery) History(_ context.Context, viewer messages.AccountID) (*chatv1.GetHistoryResponse, error) {
	*s.viewer = viewer
	return s.history, s.err
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func TestGetHistoryAnswersWhatTheQueryReadsForTheCaller(t *testing.T) {
	history := &chatv1.GetHistoryResponse{Messages: []*chatv1.ChatMessage{{Id: "message-1", Text: "hello"}}}
	viewer := new(messages.AccountID)

	res, err := get_history_handler.New(stubQuery{history: history, viewer: viewer}).
		GetHistory(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&chatv1.GetHistoryRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(history, res.Msg))
	assert.Equal(t, messages.AccountIDOf(ada), *viewer)
}

func TestACallerWithNoTokenReadsAsNobody(t *testing.T) {
	viewer := new(messages.AccountID)

	_, err := get_history_handler.New(stubQuery{history: &chatv1.GetHistoryResponse{}, viewer: viewer}).
		GetHistory(t.Context(), connect.NewRequest(&chatv1.GetHistoryRequest{}))

	require.NoError(t, err)
	assert.Equal(t, messages.NoAccount, *viewer)
}

func TestGetHistoryIsNeverCached(t *testing.T) {
	res, err := get_history_handler.New(stubQuery{history: &chatv1.GetHistoryResponse{}, viewer: new(messages.AccountID)}).
		GetHistory(t.Context(), connect.NewRequest(&chatv1.GetHistoryRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := get_history_handler.New(stubQuery{err: failure, viewer: new(messages.AccountID)}).
		GetHistory(t.Context(), connect.NewRequest(&chatv1.GetHistoryRequest{}))

	assert.ErrorIs(t, err, failure)
}
