package pay_usecase_test

import (
	"context"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

type intents map[payments.PaymentID]payments.IntentID

func (i intents) IntentOf(_ context.Context, invoice payments.PaymentID) (payments.IntentID, error) {
	return i[invoice], nil
}

type fixture struct {
	store   *payments.MemoryStore
	outbox  *cpatomic.FakeOutbox
	useCase *pay_usecase.UseCase
}

func newFixture() fixture {
	f := fixture{store: payments.NewMemoryStore(), outbox: &cpatomic.FakeOutbox{}}
	f.useCase = pay_usecase.New(cpatomic.FakeTransactioner{}, notices.NewMemoryInbox(), f.store,
		intents{"in_1": "pi_9"}, f.outbox)
	return f
}

func charged(id payments.PaymentID, plan plans.Plan, intent payments.IntentID) payments.Charged {
	return payments.Charged{
		ID: id, Order: payments.TestOrder(orders.AccountID{15: 1}, plan, now.Add(-time.Minute)), Intent: intent,
		Amount: 400, Currency: "eur", At: now,
	}
}

func TestAPaymentIsKeptAndItsTimeAskedOfTheFlags(t *testing.T) {
	f := newFixture()

	require.NoError(t, f.useCase.Execute(t.Context(), pay_usecase.In{
		Event: "evt_1", Charged: charged("cs_1", plans.Week, "pi_1"),
	}))

	require.Len(t, f.store.Kept(), 1)
	assert.Equal(t, payments.Paid, f.store.Kept()[0].State)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.GrantTime{
		GrantId: "stripe:cs_1", Period: flagsv1.TimePeriod_TIME_PERIOD_WEEK, PaidAt: timestamppb.New(now),
		Target: &flagsv1.TimeTarget{Target: &flagsv1.TimeTarget_SubmissionId{
			SubmissionId: "00000000-0000-0000-0000-000000000001",
		}},
		AccountId: orders.AccountID{15: 1}.String(), OrderedAt: timestamppb.New(now.Add(-time.Minute)),
	}, f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestAnInvoicesIntentIsReadFromStripe(t *testing.T) {
	f := newFixture()

	require.NoError(t, f.useCase.Execute(t.Context(), pay_usecase.In{
		Event: "evt_1", Charged: charged("in_1", plans.Month, ""), Invoice: true,
	}))

	assert.Equal(t, payments.IntentID("pi_9"), f.store.Kept()[0].Intent)
	assert.Equal(t, flagsv1.TimePeriod_TIME_PERIOD_MONTH, f.outbox.Recorded()[0].(*flagsv1.GrantTime).GetPeriod())
}

func TestAnEventToldAgainPaysOnce(t *testing.T) {
	f := newFixture()
	in := pay_usecase.In{Event: "evt_1", Charged: charged("cs_1", plans.Week, "pi_1")}

	require.NoError(t, f.useCase.Execute(t.Context(), in))
	require.NoError(t, f.useCase.Execute(t.Context(), in))

	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestAPaymentToldByTwoEventsPaysOnce(t *testing.T) {
	f := newFixture()

	require.NoError(t, f.useCase.Execute(t.Context(), pay_usecase.In{
		Event: "evt_1", Charged: charged("cs_1", plans.Week, "pi_1"),
	}))
	require.NoError(t, f.useCase.Execute(t.Context(), pay_usecase.In{
		Event: "evt_2", Charged: charged("cs_1", plans.Week, "pi_1"),
	}))

	assert.Len(t, f.outbox.Recorded(), 1)
}
