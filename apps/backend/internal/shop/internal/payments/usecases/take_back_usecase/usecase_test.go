package take_back_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/take_back_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

func paid(t *testing.T) (*payments.MemoryStore, *cpatomic.FakeOutbox, *take_back_usecase.UseCase) {
	t.Helper()
	store, outbox := payments.NewMemoryStore(), &cpatomic.FakeOutbox{}
	payment, _, err := payments.Pay(payments.Charged{
		ID: "cs_1", Order: payments.TestOrder(orders.AccountID{15: 1}, plans.Week, now.Add(-time.Hour)), Intent: "pi_1",
		Amount: 400, Currency: "eur", At: now.Add(-time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), payment))
	return store, outbox, take_back_usecase.New(cpatomic.FakeTransactioner{}, notices.NewMemoryInbox(), store, outbox)
}

func TestARefundAsksTheFlagsToTakeThePaymentsTimeBack(t *testing.T) {
	store, outbox, useCase := paid(t)

	require.NoError(t, useCase.Execute(t.Context(), take_back_usecase.In{Event: "evt_1", Intent: "pi_1", At: now}))

	assert.Equal(t, payments.TakenBack, store.Kept()[0].State)
	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.TakeBackTime{GrantId: "stripe:cs_1", DecidedAt: timestamppb.New(now)},
		outbox.Recorded()[0]), "%v", outbox.Recorded()[0])
}

func TestARefundThenADisputeTakeTheTimeBackOnce(t *testing.T) {
	_, outbox, useCase := paid(t)

	require.NoError(t, useCase.Execute(t.Context(), take_back_usecase.In{Event: "evt_1", Intent: "pi_1", At: now}))
	require.NoError(t, useCase.Execute(t.Context(), take_back_usecase.In{Event: "evt_2", Intent: "pi_1", At: now}))

	assert.Len(t, outbox.Recorded(), 1)
}

func TestARefundOfAnotherSaleIsNoneOfTheShops(t *testing.T) {
	_, outbox, useCase := paid(t)

	require.NoError(t, useCase.Execute(t.Context(), take_back_usecase.In{Event: "evt_1", Intent: "pi_coaching", At: now}))

	assert.Empty(t, outbox.Recorded())
}
