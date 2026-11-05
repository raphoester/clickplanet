package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestOnlyALaterDecisionOverrulesTheChoiceKept(t *testing.T) {
	at := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	kept := rules.Choice{Set: "rush", DecidedAt: at}

	assert.True(t, rules.Choice{Set: rules.Default, DecidedAt: at.Add(time.Second)}.Overrules(kept))
	assert.False(t, rules.Choice{Set: "rush", DecidedAt: at}.Overrules(kept), "sent again")
	assert.False(t, rules.Choice{Set: rules.Default, DecidedAt: at.Add(-time.Second)}.Overrules(kept), "older")
	assert.True(t, kept.Overrules(rules.NoChoice), "any decision overrules none")
}
