package planetv1controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/map_density_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/stretchr/testify/require"
)

// Nothing in this package tests ClickService itself. It is an aggregation, and
// the only claim it makes — that it carries all five procedures — is the
// compile-time assertion in click_service.go. What the tests here are about is
// the interceptor chain, which needs a served handler to be a chain at all; the
// stubs below are how they get one cheaply.

type stubService struct {
	err error
	out click_usecase.Out
}

func (s stubService) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return s.out, s.err
}

type stubChecker struct{}

func (stubChecker) MaxIndex() uint32 { return 100 }

type stubSubscriber struct {
	updates chan clicks.Change
	err     error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan clicks.Change, error) {
	return s.updates, s.err
}

type stubMapReader struct{}

func (stubMapReader) StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error) {
	if start > end {
		return clicks.DenseBatch{}, fmt.Errorf("invalid tile range")
	}

	return clicks.DenseBatch{Start: start, Codes: []string{"", "fr"}, Tiles: []byte{0x01, 0x00, 0x00, 0x00}}, nil
}

func clickServer(t *testing.T, options ...connect.HandlerOption) *httptest.Server {
	t.Helper()
	return clickServerWith(t, stubService{}, nil, options...)
}

// clickServerWith takes the click chain whole, so a test that is about the
// throttle wires a throttled one and every other test wires the bare stub.
func clickServerWith(
	t *testing.T,
	clickUseCase click_usecase.IUseCase,
	budgets get_budget_usecase.ClickBudgetReader,
	options ...connect.HandlerOption,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		ClickService{
			ClickHandler:      click_handler.New(clickUseCase),
			GetBudgetHandler:  get_budget_handler.New(get_budget_usecase.New(budgets, onePrice, buckets)),
			MapDensityHandler: map_density_handler.New(map_density_usecase.New(stubChecker{})),
			GetMapHandler:     get_map_handler.New(get_map_usecase.New(stubChecker{}, stubMapReader{})),
			ListenForEventsHandler: listen_for_events_handler.New(
				listen_for_events_usecase.New(stubSubscriber{}, listen_for_events_usecase.DefaultHeartbeat, nil)),
		},
		options...,
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server
}

type fakeLimiter struct {
	allow bool
	state cpratelimit.State
	keys  []cpratelimit.Key
}

func (l *fakeLimiter) TakeAll(_ float64, keys ...cpratelimit.Key) (bool, []cpratelimit.State) {
	l.keys = append(l.keys, keys...)

	states := make([]cpratelimit.State, len(keys))
	for i := range states {
		states[i] = l.state
	}
	return l.allow, states
}

type stubPricer clicks.Price

func (p stubPricer) Price(string) clicks.Price { return clicks.Price(p) }

var onePrice = stubPricer{Slowdown: 1}

var buckets = clicks.ThrottleConfig{}.Buckets()

type fakeRequest struct {
	connect.AnyRequest
	spec   connect.Spec
	header http.Header
}

func (r fakeRequest) Spec() connect.Spec { return r.spec }

func (r fakeRequest) Header() http.Header {
	if r.header == nil {
		return http.Header{}
	}

	return r.header
}

// clickStatus posts a click as a plain HTTP request, for the assertions that are
// about the status code a browser sees rather than the Connect error.
func clickStatus(t *testing.T, server *httptest.Server, ip string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		server.URL+planetv1connect.ClickServiceClickProcedure,
		strings.NewReader(`{"tileId":1,"countryId":"fr"}`))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("X-Real-IP", ip)

	res, err := server.Client().Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	return res.StatusCode
}

func budgetDetail(t *testing.T, err error) *planetv1.ClickBudget {
	t.Helper()

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)

	for _, detail := range connectErr.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}

		if budget, ok := value.(*planetv1.ClickBudget); ok {
			return budget
		}
	}

	t.Fatal("the refusal carried no budget")

	return nil
}

// errorNet is what cpbootstrap wraps around every service it mounts. These
// tests serve a handler directly, so they wire it themselves — the assertions
// about redaction live in cpconnect, but a chain without it would report an
// unrecognised error differently from the real server.
func errorNet() connect.Interceptor {
	return cpconnect.NewErrorInterceptor(nil, nil)
}
