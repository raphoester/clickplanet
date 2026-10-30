package get_mute_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_mute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubQuery struct {
	account messages.AccountID
	err     error
}

func (q *stubQuery) Mute(_ context.Context, account messages.AccountID) (*chatv1.GetMuteResponse, error) {
	q.account = account
	if q.err != nil {
		return nil, q.err
	}
	return &chatv1.GetMuteResponse{
		MutedUntil:     timestamppb.New(time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)),
		MutedAccountId: account.String(),
	}, nil
}

const bullyID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func TestTheMuteIsAnsweredAndNeverCached(t *testing.T) {
	query := &stubQuery{}

	res, err := get_mute_handler.New(query).GetMute(t.Context(), connect.NewRequest(&chatv1.GetMuteRequest{AccountId: bullyID}))
	require.NoError(t, err)

	assert.Equal(t, messages.AccountIDOf(bullyID), query.account)
	assert.Equal(t, bullyID, res.Msg.GetMutedAccountId())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAnIDThatIsNotAnAccountIsInvalid(t *testing.T) {
	for _, id := range []string{"", "not-an-account"} {
		_, err := get_mute_handler.New(&stubQuery{}).GetMute(t.Context(), connect.NewRequest(&chatv1.GetMuteRequest{AccountId: id}))
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
	}
}

func TestAReadThatFailsIsNoAnswer(t *testing.T) {
	_, err := get_mute_handler.New(&stubQuery{err: assert.AnError}).
		GetMute(t.Context(), connect.NewRequest(&chatv1.GetMuteRequest{AccountId: bullyID}))

	assert.ErrorIs(t, err, assert.AnError)
}
