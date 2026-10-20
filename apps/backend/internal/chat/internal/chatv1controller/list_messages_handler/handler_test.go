package list_messages_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler/messages_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubQuery struct {
	err    error
	cursor *messages_query.Cursor
	author *messages.AccountID
	limit  *uint32
}

func (q stubQuery) Messages(
	_ context.Context, cursor messages_query.Cursor, author messages.AccountID, limit uint32,
) (*chatv1.ListMessagesResponse, error) {
	if q.cursor != nil {
		*q.cursor, *q.author, *q.limit = cursor, author, limit
	}
	if q.err != nil {
		return nil, q.err
	}
	return &chatv1.ListMessagesResponse{Messages: []*chatv1.KeptMessage{{MessageId: "insult"}}}, nil
}

func asking() stubQuery {
	return stubQuery{cursor: new(messages_query.Cursor), author: new(messages.AccountID), limit: new(uint32)}
}

func TestThePageAskedForIsAnsweredAndNeverCached(t *testing.T) {
	query := asking()

	res, err := list_messages_handler.New(query).ListMessages(t.Context(),
		connect.NewRequest(&chatv1.ListMessagesRequest{BeforeMessageId: "insult", Limit: 10}))
	require.NoError(t, err)

	assert.Equal(t, messages_query.Cursor{Before: "insult"}, *query.cursor)
	assert.Equal(t, messages.NoAccount, *query.author, "no account is everyone's messages")
	assert.Equal(t, uint32(10), *query.limit)
	assert.Equal(t, "insult", res.Msg.GetMessages()[0].GetMessageId())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAnAccountAskedForIsTheOneAuthorRead(t *testing.T) {
	query := asking()
	ada := messages.AccountID{15: 1}

	_, err := list_messages_handler.New(query).ListMessages(t.Context(),
		connect.NewRequest(&chatv1.ListMessagesRequest{AccountId: ada.String()}))
	require.NoError(t, err)

	assert.Equal(t, ada, *query.author)
}

func TestAnAccountThatIsNotAUUIDIsRefusedNotReadAsEveryone(t *testing.T) {
	query := asking()

	_, err := list_messages_handler.New(query).ListMessages(t.Context(),
		connect.NewRequest(&chatv1.ListMessagesRequest{AccountId: "Ada"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Equal(t, messages_query.Cursor{}, *query.cursor, "nothing was read")
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		messages_query.ErrTwoCursors:                                connect.CodeInvalidArgument,
		fmt.Errorf("%w: %q", messages_query.ErrNoMessage, "pruned"): connect.CodeNotFound,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := list_messages_handler.New(stubQuery{err: err}).
			ListMessages(t.Context(), connect.NewRequest(&chatv1.ListMessagesRequest{AfterMessageId: "pruned"}))
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
