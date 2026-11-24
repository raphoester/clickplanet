package shop_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices/postgres_inbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices/usecases/receive_notice_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/postgres_payment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/take_back_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/postgres_renewal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/usecases/hear_renewal_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/stripe_gateway"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/stripecontroller/receive_notice_handler"
)

const (
	founder    = "00000000-0000-0000-0000-00000000000a"
	submission = "00000000-0000-0000-0000-000000000001"
)

// told keeps what the flags module is told, as its command handlers would.
type told struct {
	mu    sync.Mutex
	heard []string
}

type grants struct{ told *told }

func (g grants) Handle(_ context.Context, command *flagsv1.GrantTime) error {
	g.told.mu.Lock()
	defer g.told.mu.Unlock()
	g.told.heard = append(g.told.heard, "grant "+command.GetGrantId()+" "+command.GetPeriod().String()+" "+
		command.GetTarget().GetSubmissionId())
	return nil
}

type takebacks struct{ told *told }

func (b takebacks) Handle(_ context.Context, command *flagsv1.TakeBackTime) error {
	b.told.mu.Lock()
	defer b.told.mu.Unlock()
	b.told.heard = append(b.told.heard, "take back "+command.GetGrantId())
	return nil
}

func (t *told) all() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.heard)
}

// An invoice's intent, as Stripe's invoice payments would answer it.
type intents struct{}

func (intents) IntentOf(context.Context, payments.PaymentID) (payments.IntentID, error) {
	return "pi_renewal", nil
}

func running(t *testing.T, runners ...cpbootstrap.Runner) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	for _, runner := range runners {
		wg.Go(func() { runner.Run(ctx) })
	}
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
}

func post(t *testing.T, endpoint http.Handler, payload []byte, signature string) int {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, receive_notice_handler.Path,
		bytes.NewReader(payload))
	request.Header.Set(receive_notice_handler.SignatureHeader, signature)
	recorder := httptest.NewRecorder()
	endpoint.ServeHTTP(recorder, request)
	return recorder.Code
}

// Stripe's signed events reach the flags module as commands, through the shop's outbox: a week paid, a renewal told
// twice, a refund, and a forged event refused.
func TestStripesEventsReachTheFlagsModuleAsCommandsOnce(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "shop", migrations.FS)
	redis := cpredistest.StartServer(t).Config()
	logger := slog.New(slog.DiscardHandler)
	streams := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, streams.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = streams.Close() })
	flags := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, flags.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = flags.Close() })

	heard := &told{}
	granted, err := cpredis.Handle[*flagsv1.GrantTime](t.Context(), flags, grants{told: heard})
	require.NoError(t, err)
	takenBack, err := cpredis.Handle[*flagsv1.TakeBackTime](t.Context(), flags, takebacks{told: heard})
	require.NoError(t, err)
	running(t, granted, takenBack, cpoutbox.NewRelay("shop-outbox", db, streams.Commands(), logger))

	paid, inbox := postgres_payment_store.New(db), postgres_inbox.New(db)
	endpoint := receive_notice_handler.New(stripe_gateway.NewReader(stripe_gateway.TestSecret),
		receive_notice_usecase.New(
			pay_usecase.New(db, inbox, paid, intents{}, cpoutbox.New(db)),
			take_back_usecase.New(db, inbox, paid, cpoutbox.New(db)),
			hear_renewal_usecase.New(db, inbox, postgres_renewal_store.New(db)),
		), 1<<20)

	now := time.Now()
	week, signature := stripe_gateway.Signed("evt_week", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_week", "pi_week", stripe_gateway.Metadata(founder, submission, "week")), now)
	assert.Equal(t, http.StatusOK, post(t, endpoint, week, signature))

	renewal, signature := stripe_gateway.Signed("evt_renewal", stripe.EventTypeInvoicePaid,
		stripe_gateway.Invoice("in_renewal", "sub_1", stripe_gateway.Metadata(founder, submission, "month")), now)
	assert.Equal(t, http.StatusOK, post(t, endpoint, renewal, signature))
	assert.Equal(t, http.StatusOK, post(t, endpoint, renewal, signature), "Stripe sends an event again")

	refund, signature := stripe_gateway.Signed("evt_refund", stripe.EventTypeChargeRefunded,
		stripe_gateway.Charge("pi_renewal", true), now)
	assert.Equal(t, http.StatusOK, post(t, endpoint, refund, signature))

	forged, _ := stripe_gateway.Signed("evt_forged", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_forged", "pi_forged", stripe_gateway.Metadata(founder, submission, "year")), now)
	assert.Equal(t, http.StatusBadRequest, post(t, endpoint, forged, "t=1,v1=forged"))

	require.Eventually(t, func() bool { return len(heard.all()) == 3 }, 10*time.Second, 20*time.Millisecond)
	assert.ElementsMatch(t, []string{
		"grant stripe:cs_week TIME_PERIOD_WEEK " + submission,
		"grant stripe:in_renewal TIME_PERIOD_MONTH " + submission,
		"take back stripe:in_renewal",
	}, heard.all())
}
