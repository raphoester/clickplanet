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
	encoded := chatmessage.Encode(messages.NewMessage("message-1", time.Time{}, messages.NoAccount, "", "").Named(messages.AuthorOf("Ada_L", false, int32(playerv1.NameColor_NAME_COLOR_PINK), 12, messages.Title{})), nil, 0)

	assert.Equal(t, playerv1.NameColor_NAME_COLOR_PINK, encoded.GetAuthorColor())
	assert.Equal(t, uint32(12), encoded.GetAuthorStreak())
}

func TestAMessageCarriesTheTitleItsAuthorWears(t *testing.T) {
	settler := messages.TitleOf("settler", "Settler", messages.RankOf("conquest", "Conquest", 1, 5))

	ranked := chatmessage.Encode(messages.NewMessage("message-1", time.Time{}, messages.NoAccount, "", "").Named(messages.AuthorOf("", false, 0, 0, settler)), nil, 0)
	standalone := chatmessage.Encode(messages.NewMessage("message-2", time.Time{}, messages.NoAccount, "", "").Named(messages.AuthorOf("", false, 0, 0, messages.TitleOf("og", "OG", messages.Rank{}))), nil, 0)
	bare := chatmessage.Encode(messages.NewMessage("message-3", time.Time{}, messages.NoAccount, "", ""), nil, 0)

	assert.Equal(t, "settler", ranked.GetAuthorTitle().GetId())
	assert.Equal(t, "conquest", ranked.GetAuthorTitle().GetRank().GetTrackId())
	assert.Equal(t, uint32(5), ranked.GetAuthorTitle().GetRank().GetCount())
	assert.Equal(t, "OG", standalone.GetAuthorTitle().GetName())
	assert.Nil(t, standalone.GetAuthorTitle().GetRank())
	assert.Nil(t, bare.GetAuthorTitle())
}

func TestOneNameOverTheCapIsCutToo(t *testing.T) {
	names := make([]string, 0, chatmessage.NamedReactors+1)
	for i := range cap(names) {
		names = append(names, fmt.Sprintf("player%d", i))
	}

	encoded := chatmessage.EncodeCounts([]reactions.Count{reactions.CountOf(clown, len(names), false, nil, names)})

	assert.Equal(t, names[:chatmessage.NamedReactors], encoded[0].GetReactors())
}
