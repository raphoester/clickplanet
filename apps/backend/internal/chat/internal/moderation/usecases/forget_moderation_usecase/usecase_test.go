package forget_moderation_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/inmemory_moderation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/usecases/forget_moderation_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

var (
	at       = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	ada      = messages.AccountID{15: 1}
	bob      = messages.AccountID{15: 2}
	operator = messages.AccountID{15: 9}
)

func TestTheActionsTakenOnTheAccountGoAndAnOperatorIsTakenOffItsOwn(t *testing.T) {
	history := inmemory_moderation_storage.New()
	by := moderation.OperatorOf(operator, "Claude")
	require.NoError(t, history.Record(t.Context(), moderation.NewUnmute(ada, mutes.Chat, at, "served", by)))
	require.NoError(t, history.Record(t.Context(), moderation.NewUnmute(bob, mutes.Gifs, at, "served", by)))
	require.NoError(t, history.Record(t.Context(), moderation.NewUnmute(operator, mutes.Chat, at, "", by)))

	require.NoError(t, forget_moderation_usecase.New(history).Execute(t.Context(), ada))
	require.NoError(t, forget_moderation_usecase.New(history).Execute(t.Context(), operator))

	assert.Equal(t, []moderation.Snapshot{{
		Account: bob, At: at, Kind: moderation.Unmuted, MuteKind: mutes.Gifs, Reason: "served",
		By: moderation.OperatorOf(messages.NoAccount, "Claude"),
	}}, history.Kept())
}

func TestAFailureIsAnError(t *testing.T) {
	history := inmemory_moderation_storage.New()
	history.FailWith(errors.New("postgres is down"))

	require.Error(t, forget_moderation_usecase.New(history).Execute(t.Context(), ada))
}
