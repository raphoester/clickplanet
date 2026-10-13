package chatmessage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

const clown = reactions.Reaction(2)

func TestACountCarriesWhoGaveItOldestFirst(t *testing.T) {
	encoded := chatmessage.EncodeCounts([]reactions.Count{
		reactions.CountOf(clown, 2, true, nil, []string{"Ada", "Bo"}),
	})

	assert.Equal(t, []string{"Ada", "Bo"}, encoded[0].GetReactors())
	assert.Equal(t, uint32(2), encoded[0].GetCount())
	assert.Equal(t, chatv1.Reaction(clown), encoded[0].GetReaction())
	assert.True(t, encoded[0].GetMine())
}

func TestTheNamesAreCutAtTheCapAndTheCountIsNot(t *testing.T) {
	names := make([]string, 0, chatmessage.NamedReactors+5)
	for i := range cap(names) {
		names = append(names, fmt.Sprintf("player%d", i))
	}

	encoded := chatmessage.EncodeCounts([]reactions.Count{
		reactions.CountOf(clown, len(names), false, nil, names),
	})

	assert.Len(t, encoded[0].GetReactors(), chatmessage.NamedReactors)
	assert.Equal(t, names[:chatmessage.NamedReactors], encoded[0].GetReactors())
	assert.Equal(t, uint32(len(names)), encoded[0].GetCount(), "how many gave it, named or not")
}

func TestACountNobodyIsNamedOnCarriesNoReactors(t *testing.T) {
	encoded := chatmessage.EncodeCounts([]reactions.Count{reactions.CountOf(clown, 1, false, nil, nil)})

	assert.Empty(t, encoded[0].GetReactors(), "a reaction from before names were kept is only counted")
}

func TestAMessageCarriesItsAuthorsColorAndStreak(t *testing.T) {
	encoded := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-1", Author: messages.AuthorOf("Ada_L", false, int32(playerv1.NameColor_NAME_COLOR_PINK), 12, messages.Title{}),
	}), nil, 0)

	assert.Equal(t, playerv1.NameColor_NAME_COLOR_PINK, encoded.GetAuthorColor())
	assert.Equal(t, uint32(12), encoded.GetAuthorStreak())
}

func TestAMessageCarriesTheTitleItsAuthorWears(t *testing.T) {
	settler := messages.TitleOf("settler", "Settler", messages.RankOf("conquest", "Conquest", 1, 5))

	og := messages.TitleOf("og", "OG", messages.Rank{})

	ranked := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-1", Author: messages.AuthorOf("", false, 0, 0, settler),
	}), nil, 0)
	standalone := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-2", Author: messages.AuthorOf("", false, 0, 0, og),
	}), nil, 0)
	bare := chatmessage.Encode(messages.Restored(t, messages.Snapshot{ID: "message-3"}), nil, 0)

	assert.Equal(t, "settler", ranked.GetAuthorTitle().GetId())
	assert.Equal(t, "conquest", ranked.GetAuthorTitle().GetRank().GetTrackId())
	assert.Equal(t, uint32(5), ranked.GetAuthorTitle().GetRank().GetCount())
	assert.Equal(t, "OG", standalone.GetAuthorTitle().GetName())
	assert.Nil(t, standalone.GetAuthorTitle().GetRank())
	assert.Nil(t, bare.GetAuthorTitle())
}

func TestAMessageCarriesWhatItsSnapshotHolds(t *testing.T) {
	at := time.Date(2026, 10, 11, 12, 0, 0, 123_000_000, time.UTC)

	ada := messages.AccountID{15: 1}

	encoded := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-1", SentAt: at, Account: ada, Author: messages.AuthorOf("Ada_L", true, 0, 0, messages.Title{}),
		Country: "fr", Text: "hello",
	}), nil, 0)

	assert.Equal(t, "message-1", encoded.GetId())
	assert.Equal(t, at.UnixMilli(), encoded.GetSentAtUnixMs())
	assert.Equal(t, ada.String(), encoded.GetAuthorAccountId())
	assert.Equal(t, "Ada_L", encoded.GetAuthorName())
	assert.True(t, encoded.GetAuthorAdmin())
	assert.Equal(t, "fr", encoded.GetCountryId())
	assert.Equal(t, "hello", encoded.GetText())
}

func TestAMessageWithNoAccountCarriesNoAccountID(t *testing.T) {
	encoded := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-1", SentAt: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC),
		Author: messages.AuthorOf("guest_Bob", false, 0, 0, messages.Title{}), Country: "fr", Text: "hello",
	}), nil, 0)

	assert.Empty(t, encoded.GetAuthorAccountId(), "not an id of zeros")
}

func TestAReplyCarriesTheMessageItQuotesAndAPlainMessageNone(t *testing.T) {
	ada := messages.AuthorOf("Ada_L", true, int32(playerv1.NameColor_NAME_COLOR_PINK), 12, messages.Title{})

	reply := chatmessage.Encode(messages.Restored(t, messages.Snapshot{
		ID: "message-2", Quote: messages.Quote{MessageID: "message-1", Author: ada, Text: "who holds the north"},
	}), nil, 0)
	plain := chatmessage.Encode(messages.Restored(t, messages.Snapshot{ID: "message-3"}), nil, 0)

	assert.Equal(t, "message-1", reply.GetReplyTo().GetMessageId())
	assert.Equal(t, "Ada_L", reply.GetReplyTo().GetAuthorName())
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_PINK, reply.GetReplyTo().GetAuthorColor())
	assert.Equal(t, "who holds the north", reply.GetReplyTo().GetText())
	assert.Nil(t, plain.GetReplyTo())
}

func TestOneNameOverTheCapIsCutToo(t *testing.T) {
	names := make([]string, 0, chatmessage.NamedReactors+1)
	for i := range cap(names) {
		names = append(names, fmt.Sprintf("player%d", i))
	}

	encoded := chatmessage.EncodeCounts([]reactions.Count{reactions.CountOf(clown, len(names), false, nil, names)})

	assert.Equal(t, names[:chatmessage.NamedReactors], encoded[0].GetReactors())
}
