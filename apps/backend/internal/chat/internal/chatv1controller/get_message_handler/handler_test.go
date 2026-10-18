package get_message_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler/message_query"
)

type stubQuery struct {
	err error
}

func (q stubQuery) Message(_ context.Context, id string) (*chatv1.GetMessageResponse, error) {
	if q.err != nil {
		return nil, q.err
	}
	return &chatv1.GetMessageResponse{MessageId: id, Text: "the text of " + id}, nil
}

func TestTheMessageIsAnsweredAndNeverCached(t *testing.T) {
	res, err := get_message_handler.New(stubQuery{}).
		GetMessage(t.Context(), connect.NewRequest(&chatv1.GetMessageRequest{MessageId: "insult"}))
	require.NoError(t, err)

	assert.Equal(t, "the text of insult", res.Msg.GetText())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", message_query.ErrNoMessage, "pruned"): connect.CodeNotFound,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := get_message_handler.New(stubQuery{err: err}).
			GetMessage(t.Context(), connect.NewRequest(&chatv1.GetMessageRequest{MessageId: "pruned"}))
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
