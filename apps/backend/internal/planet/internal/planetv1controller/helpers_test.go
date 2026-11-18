package planetv1controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubService struct {
	err error
	out click_usecase.Out
}

func (s stubService) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return s.out, s.err
}

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
			ClickHandler:     click_handler.New(clickUseCase),
			GetBudgetHandler: get_budget_handler.New(get_budget_usecase.New(budgets, onePrice, buckets, thawed)),
		},
		options...,
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server
}

type stubPricer clicks.Price

func (p stubPricer) Price(string) clicks.Price { return clicks.Price(p) }

var onePrice = stubPricer{Slowdown: 1}

// wholeClicks prices every click a whole one: no rally anywhere.
type wholeClicks struct{}

func (wholeClicks) Cost(context.Context, string, uint32) (float64, error) { return 1, nil }

var buckets = clicks.NewBuckets(clicks.TestThrottle, cptime.SystemClock{})

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

func errorNet() connect.Interceptor {
	return cpconnect.NewErrorInterceptor(nil)
}

func thawed() clicks.Freeze { return false }
