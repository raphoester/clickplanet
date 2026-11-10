package chatmessage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

const clown = reactions.Reaction(2)

func reactors(how int) ([]reactions.Reactor, []string) {
	gave := make([]reactions.Reactor, 0, how)
	ids := make([]string, 0, how)
	for i := range how {
		account := messages.AccountID{14: 1, 15: byte(i)}
		gave = append(gave, reactions.ReactorOf(account))
		ids = append(ids, account.String())
	}
	return gave, ids
}

func TestACountCarriesTheAccountsThatGaveItOldestFirst(t *testing.T) {
	gave, ids := reactors(2)

	encoded := chatmessage.EncodeCounts([]reactions.Count{reactions.CountOf(clown, 2, true, gave)})

	assert.Equal(t, ids, encoded[0].GetReactorAccountIds())
	assert.Equal(t, uint32(2), encoded[0].GetCount())
	assert.Equal(t, chatv1.Reaction(clown), encoded[0].GetReaction())
	assert.True(t, encoded[0].GetMine())
}

func TestTheAccountsAreCutAtTheCapAndTheCountIsNot(t *testing.T) {
	for _, how := range []int{chatmessage.NamedReactors + 1, chatmessage.NamedReactors + 5} {
		gave, ids := reactors(how)

		encoded := chatmessage.EncodeCounts([]reactions.Count{reactions.CountOf(clown, how, false, gave)})

		assert.Equal(t, ids[:chatmessage.NamedReactors], encoded[0].GetReactorAccountIds(), how)
		assert.Equal(t, uint32(how), encoded[0].GetCount(), "how many gave it, carried or not")
	}
}

func TestAGuestTagFromBeforeAccountsIsOnlyCounted(t *testing.T) {
	encoded := chatmessage.EncodeCounts([]reactions.Count{reactions.CountOf(clown, 1, false, []reactions.Reactor{"guest:91aa3d"})})

	assert.Empty(t, encoded[0].GetReactorAccountIds())
	assert.Equal(t, uint32(1), encoded[0].GetCount())
}

func TestAMessageCarriesWhatItsSnapshotHolds(t *testing.T) {
	at := time.Date(2026, 10, 11, 12, 0, 0, 123_000_000, time.UTC)
	ada := messages.AccountID{15: 1}

	encoded := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-1", SentAt: at, Account: ada, Country: "fr", Text: "hello",
	}))

	assert.Equal(t, "message-1", encoded.GetId())
	assert.Equal(t, at.UnixMilli(), encoded.GetSentAtUnixMs())
	assert.Equal(t, ada.String(), encoded.GetAuthorAccountId(), "the page names it")
	assert.Equal(t, "fr", encoded.GetCountryId())
	assert.Equal(t, "hello", encoded.GetText())
}

func TestAMessageWithNoAccountCarriesNoAccountID(t *testing.T) {
	encoded := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-1", SentAt: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC), Country: "fr", Text: "hello",
	}))

	assert.Empty(t, encoded.GetAuthorAccountId(), "not an id of zeros")
}

func TestAReplyCarriesTheMessageItQuotesAndAPlainMessageNone(t *testing.T) {
	ada := messages.AccountID{15: 1}

	reply := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-2", Quote: messages.Quote{MessageID: "message-1", Account: ada, Text: "who holds the north"},
	}))
	plain := chatmessage.Encode(messages.Restored(t, messages.Snapshot{ID: "message-3"}))

	assert.Equal(t, "message-1", reply.GetReplyTo().GetMessageId())
	assert.Equal(t, ada.String(), reply.GetReplyTo().GetAuthorAccountId())
	assert.Equal(t, "who holds the north", reply.GetReplyTo().GetText())
	assert.Nil(t, plain.GetReplyTo())
}
