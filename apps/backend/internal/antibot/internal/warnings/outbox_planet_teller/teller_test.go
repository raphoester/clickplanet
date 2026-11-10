package outbox_planet_teller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings/outbox_planet_teller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var at = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

const ada = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func TestAWarningIsAnAccountPlanetWarns(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	require.NoError(t, cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
		return outbox_planet_teller.New(outbox).Tell(ctx, warnings.Warning{Account: ada, At: at})
	}))

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&planetv1.WarnCaller{AccountId: ada, DecidedAt: timestamppb.New(at)}, outbox.Recorded()[0]))
}

func TestAWarningIsNeverToldOutsideItsTransaction(t *testing.T) {
	err := outbox_planet_teller.New(&cpatomic.FakeOutbox{}).Tell(t.Context(), warnings.Warning{Account: ada, At: at})

	assert.ErrorIs(t, err, cpatomic.ErrNoTransaction)
}
