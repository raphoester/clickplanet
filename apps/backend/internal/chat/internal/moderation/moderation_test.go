package moderation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

var (
	at     = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	bob    = messages.AccountID{15: 2}
	claude = moderation.OperatorOf(messages.AccountID{15: 9}, "Claude")
)

func TestAMuteIsTheAccountsAndNeverItsNetworks(t *testing.T) {
	mute := mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bob, "2a01:e0a:1:2::/64"), mutes.Gifs, at, time.Hour)

	assert.Equal(t, moderation.Snapshot{
		Account: bob, At: at, Kind: moderation.Muted, MuteKind: mutes.Gifs, Duration: time.Hour, Reason: "spam",
		By: claude,
	}, moderation.NewMute(mute, "spam", claude).TakeSnapshot())
}

func TestARemovalCopiesWhatTheMessageSaidAndItsAuthor(t *testing.T) {
	message := messages.Restored(t, messages.Snapshot{
		ID: "insult", SentAt: at.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck",
	})

	assert.Equal(t, moderation.Snapshot{
		Account: bob, At: at, Kind: moderation.Removed, Message: "insult", Text: "you all suck", By: claude,
	}, moderation.NewRemoval(message, at, moderation.NoReason, claude).TakeSnapshot())
}

func TestEachKindIsAValueOfThePostgresType(t *testing.T) {
	for kind, kept := range map[moderation.Kind]string{
		moderation.Muted: "mute", moderation.Unmuted: "unmute", moderation.Removed: "removal",
	} {
		assert.Equal(t, kept, kind.String())
	}
}
