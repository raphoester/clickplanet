package redis_outbox_sender_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/redis_outbox_sender"
)

type sender struct {
	sent []proto.Message
	err  error
}

func (s *sender) Send(_ context.Context, messages ...proto.Message) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, messages...)
	return nil
}

var (
	grant = &planetv1.GrantCharges{GrantId: "g1", AccountId: "ada"}
	other = &planetv1.GrantCharges{GrantId: "g2", AccountId: "ada"}
	paid  = &missionsv1.MissionsPaid{AccountId: "ada", PaidInAll: 2}
	flag  = &scoringv1.GrantPoints{GrantId: "f1", Country: "fr", Points: 2}
)

func TestTheGrantsGoToTheirHandlersAndTheMissionsPaidToEveryListener(t *testing.T) {
	events, commands := &sender{}, &sender{}

	require.NoError(t, redis_outbox_sender.New(events, commands).Send(t.Context(), grant, flag, other, paid))

	assert.Equal(t, []proto.Message{grant, flag, other}, commands.sent, "in the order kept")
	assert.Equal(t, []proto.Message{paid}, events.sent)
}

func TestAMessageTheOutboxDoesNotKeepSendsNothing(t *testing.T) {
	events, commands := &sender{}, &sender{}

	err := redis_outbox_sender.New(events, commands).Send(t.Context(), grant, &planetv1.RefuseCaller{})

	require.Error(t, err)
	assert.Empty(t, commands.sent)
	assert.Empty(t, events.sent)
}

func TestAFailedSendIsAnError(t *testing.T) {
	down := errors.New("down")

	assert.ErrorIs(t, redis_outbox_sender.New(&sender{err: down}, &sender{}).Send(t.Context(), grant, paid), down)
	assert.ErrorIs(t, redis_outbox_sender.New(&sender{}, &sender{err: down}).Send(t.Context(), grant, paid), down)
}
