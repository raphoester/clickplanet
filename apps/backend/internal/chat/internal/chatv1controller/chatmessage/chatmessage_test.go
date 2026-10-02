package chatmessage_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

const clown = reactions.Reaction(2)

func TestACountCarriesWhoGaveItOldestFirst(t *testing.T) {
	encoded := chatmessage.EncodeCounts([]reactions.Count{
		{Reaction: clown, Count: 2, Mine: true, Names: []string{"Ada", "Bo"}},
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
		{Reaction: clown, Count: len(names), Mine: false, Names: names},
	})

	assert.Len(t, encoded[0].GetReactors(), chatmessage.NamedReactors)
	assert.Equal(t, names[:chatmessage.NamedReactors], encoded[0].GetReactors())
	assert.Equal(t, uint32(len(names)), encoded[0].GetCount(), "how many gave it, named or not")
}

func TestACountNobodyIsNamedOnCarriesNoReactors(t *testing.T) {
	encoded := chatmessage.EncodeCounts([]reactions.Count{{Reaction: clown, Count: 1}})

	assert.Empty(t, encoded[0].GetReactors(), "a reaction from before names were kept is only counted")
}
