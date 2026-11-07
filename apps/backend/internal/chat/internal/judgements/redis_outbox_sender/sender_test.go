package redis_outbox_sender_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/redis_outbox_sender"
)

type recorder struct {
	sent []proto.Message
	err  error
}

func (r *recorder) Send(_ context.Context, messages ...proto.Message) error {
	r.sent = append(r.sent, messages...)
	return r.err
}

func TestADecisionIsACommandAndAMuteAnEvent(t *testing.T) {
	events, commands := &recorder{}, &recorder{}
	decision, muted := &supportv1.DecideReport{ReportId: "r-1"}, &chatv1.AccountMuted{MuteId: "m-1"}

	require.NoError(t, redis_outbox_sender.New(events, commands).Send(t.Context(), muted, decision))

	assert.Equal(t, []proto.Message{muted}, events.sent)
	assert.Equal(t, []proto.Message{decision}, commands.sent)
}

func TestAMessageTheOutboxDoesNotKeepIsRefusedAndNothingIsSent(t *testing.T) {
	events, commands := &recorder{}, &recorder{}

	err := redis_outbox_sender.New(events, commands).Send(t.Context(), &supportv1.DecideReport{}, wrapperspb.String("x"))

	require.Error(t, err)
	assert.Empty(t, events.sent)
	assert.Empty(t, commands.sent)
}

func TestARefusedSendFails(t *testing.T) {
	cause := errors.New("redis is down")

	err := redis_outbox_sender.New(&recorder{}, &recorder{err: cause}).Send(t.Context(), &supportv1.DecideReport{})

	require.ErrorIs(t, err, cause)
}
