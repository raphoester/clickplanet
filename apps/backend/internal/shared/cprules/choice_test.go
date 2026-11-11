package cprules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestOnlyALaterDecisionOverrulesTheChoiceKept(t *testing.T) {
	at := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	kept := cprules.Choice{Set: "rush", DecidedAt: at}

	assert.True(t, cprules.Choice{Set: cprules.Default, DecidedAt: at.Add(time.Second)}.Overrules(kept))
	assert.False(t, cprules.Choice{Set: "rush", DecidedAt: at}.Overrules(kept), "sent again")
	assert.False(t, cprules.Choice{Set: cprules.Default, DecidedAt: at.Add(-time.Second)}.Overrules(kept), "older")
	assert.True(t, kept.Overrules(cprules.NoChoice), "any decision overrules none")
}
