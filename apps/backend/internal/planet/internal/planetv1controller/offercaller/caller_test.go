package offercaller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/offercaller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func TestTheCallerIsTheTokensAccountAndTheAddresssScope(t *testing.T) {
	guest := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")

	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest"}, offercaller.Of(guest))
	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest", Linked: true},
		offercaller.Of(cpctx.AddLinkedToContext(guest)))
	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4"}, offercaller.Of(cpctx.AddIPToContext(t.Context(), "1.2.3.4")))
}
