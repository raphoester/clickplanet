package message_sent_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/message_sent_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

func TestAMessageIsCountedOnItsSendersStats(t *testing.T) {
	store := inmemory_player_store.New()

	err := message_sent_subscriber.New(record_message_usecase.New(store)).Handle(t.Context(),
		&chatv1.MessageSent{MessageId: "m1", AccountId: account, SentAt: timestamppb.New(at)})

	require.NoError(t, err)
	id, err := players.AccountIDOf(account)
	require.NoError(t, err)
	stats, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), stats.MessagesSent())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := message_sent_subscriber.New(record_message_usecase.New(inmemory_player_store.New())).Handle(t.Context(),
		&chatv1.MessageSent{MessageId: "m1", SentAt: timestamppb.New(at)})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
