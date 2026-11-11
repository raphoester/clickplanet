package caller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func TestTheCallerIsTheTokensAccountAndTheAddresssScope(t *testing.T) {
	guest := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest"}, caller.Of(guest))
	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest", Linked: true},
		caller.Of(cpctx.AddLinkedToContext(guest)))
	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4"}, caller.Of(cpctx.AddIPToContext(t.Context(), "1.2.3.4")))
}
