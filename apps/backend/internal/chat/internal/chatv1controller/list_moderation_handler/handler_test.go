package list_moderation_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler/moderation_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubQuery struct {
	asked *moderation_query.Filter
}

func (s *stubQuery) Actions(_ context.Context, filter moderation_query.Filter) (*chatv1.ListModerationResponse, error) {
	s.asked = &filter
	return &chatv1.ListModerationResponse{}, nil
}

const bobID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func list(t *testing.T, query *stubQuery, req *chatv1.ListModerationRequest) (*connect.Response[chatv1.ListModerationResponse], error) {
	t.Helper()

	return list_moderation_handler.New(query).ListModeration(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the tests read the code.
}

func TestTheFilterReachesTheQueryAndTheAnswerIsNeverCached(t *testing.T) {
	since := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	query := &stubQuery{}

	res, err := list(t, query, &chatv1.ListModerationRequest{AccountId: bobID, Since: timestamppb.New(since), Limit: 7})
	require.NoError(t, err)

	assert.Equal(t, &moderation_query.Filter{Account: messages.AccountIDOf(bobID), Since: since, Limit: 7}, query.asked)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestNoAccountIsEveryAccountAndNoTimeTheWholeHistory(t *testing.T) {
	query := &stubQuery{}

	_, err := list(t, query, &chatv1.ListModerationRequest{})
	require.NoError(t, err)

	assert.Equal(t, messages.NoAccount, query.asked.Account)
	assert.Equal(t, time.Unix(0, 0), query.asked.Since)
}

func TestAnAccountThatIsNotAUUIDIsInvalidAndNothingIsRead(t *testing.T) {
	query := &stubQuery{}

	_, err := list(t, query, &chatv1.ListModerationRequest{AccountId: "Bob"})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, query.asked)
}
