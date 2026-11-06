package prune_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/prune_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/inmemory_reaction_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

type failingPruner struct{}

func (failingPruner) DeleteBefore(context.Context, time.Time) (int64, error) {
	return 0, errors.New("postgres is down")
}

func TestAPruneDeletesMessagesReactionsAndAnnouncementsPastRetention(t *testing.T) {
	sent := inmemory_message_storage.New()
	given := inmemory_reaction_storage.New()
	announced := inmemory_announcement_storage.New()
	for _, at := range []time.Time{now.Add(-48 * time.Hour), now.Add(-time.Hour)} {
		id := messages.MessageID(at.String())
		require.NoError(t, sent.Append(t.Context(),
			messages.NewRecord(messages.Restored(t, messages.Snapshot{ID: id, SentAt: at}), "", "", "")))
		require.NoError(t, given.Save(t.Context(), reactions.On(id, 1, "guest:a", at)))
		_, err := announced.AppendOnce(t.Context(), announcements.NewAnnouncement(announcements.AnnouncementID(uuid.New()), announcements.KindBomb, at, nil))
		require.NoError(t, err)
	}

	deleted, err := prune_usecase.New(cptime.NewFixedClock(now),
		prune_usecase.KeptFor(24*time.Hour, sent, given, announced)).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	assert.Len(t, sent.Kept(), 1)
}

func TestTheModerationHistoryIsPrunedPastARetentionOfItsOwn(t *testing.T) {
	sent := inmemory_message_storage.New()
	history := inmemory_moderation_storage.New()
	for _, at := range []time.Time{now.Add(-100 * 24 * time.Hour), now.Add(-48 * time.Hour)} {
		require.NoError(t, sent.Append(t.Context(), messages.NewRecord(
			messages.Restored(t, messages.Snapshot{ID: messages.MessageID(at.String()), SentAt: at}), "", "", "")))
		require.NoError(t, history.Record(t.Context(),
			moderation.NewUnmute(messages.AccountID{15: 1}, mutes.Chat, at, "", moderation.Operator{})))
	}

	deleted, err := prune_usecase.New(cptime.NewFixedClock(now),
		prune_usecase.KeptFor(24*time.Hour, sent), prune_usecase.KeptFor(90*24*time.Hour, history)).Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int64(3), deleted)
	assert.Empty(t, sent.Kept())
	require.Len(t, history.Kept(), 1, "the history outlives the chat's retention")
	assert.Equal(t, now.Add(-48*time.Hour), history.Kept()[0].At)
}

func TestAFailedPruneSaysSo(t *testing.T) {
	_, err := prune_usecase.New(cptime.NewFixedClock(now), prune_usecase.KeptFor(time.Hour, failingPruner{})).
		Execute(t.Context())

	require.Error(t, err)
}
