package rule_set_message_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
)

func TestASetIsTheSameAfterItsMessage(t *testing.T) {
	rush := rules.TestSnapshot("rush")
	rush.Bonuses.Quiz.Enabled = true

	assert.Equal(t, rush, rule_set_message.Decode(rule_set_message.Encode(rush)))
}

func TestASetIsTheSameAfterItsDocument(t *testing.T) {
	rush := rules.TestSnapshot("rush")

	document, err := rule_set_message.Document(rush)
	require.NoError(t, err)
	message, err := rule_set_message.Parse("rush", document)
	require.NoError(t, err)

	assert.Equal(t, rush, rule_set_message.Decode(message))
	assert.NotContains(t, string(document), `"name"`, "the row names the set")
}

func TestADocumentWithAValueThisBuildDoesNotKnowIsRefused(t *testing.T) {
	_, err := rule_set_message.Parse("rush", []byte(`{"tempo": 1, "frozen": true}`))

	require.ErrorContains(t, err, "rush")
}

func TestASetWithNoTollKeepsNone(t *testing.T) {
	free := rules.TestSnapshot("free")
	free.Toll = nil

	assert.Nil(t, rule_set_message.Decode(rule_set_message.Encode(free)).Toll)
}
