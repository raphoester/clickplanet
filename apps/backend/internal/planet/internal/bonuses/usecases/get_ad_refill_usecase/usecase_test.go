package get_ad_refill_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/get_ad_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func TestTheCallersBankIsRead(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 30, 12, 34, 0, 0, time.UTC))
	refills := bonuses.NewAdRefills(bonuses.NewMemoryGrants(), bonuses.TestRules, clock)
	useCase := get_ad_refill_usecase.New(refills)
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), ada)
	require.NoError(t, refills.Record(t.Context(), clicks.Payer{Scope: "1.2.3.4", Account: ada}))

	at, err := useCase.Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC), at)
}
