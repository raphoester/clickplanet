package planetv1controller

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
)

type ClickBudget = connectutil.ClickBudget

// ErrBudgetSpent reads like ErrNoSession on purpose, and reaches the player as
// little: the client mints another token and retries the click. What separates
// them is only visible here and in the counter below.
var ErrBudgetSpent = errors.New("this session has spent its clicks; mint another one")

func NewClickBudgetInterceptor(
	budget ClickBudget,
	registerer prometheus.Registerer,
) (connect.Interceptor, error) {
	// Unlabelled, and worth watching on its own: a session that spends a whole
	// budget is a session that clicked at the throttle, without pause, for as
	// long as the budget covers. Players do not do that. A rate here that is
	// not near zero is the shape of automation, not of a busy day.
	exhausted := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "click_budget_exhausted",
		Help: "Clicks refused because the session that carried them had spent its budget",
	})

	if err := registerer.Register(exhausted); err != nil {
		return nil, fmt.Errorf("failed to register counter: %w", err)
	}

	return connectutil.NewClickBudgetInterceptor(
		budget,
		ErrBudgetSpent,
		exhausted.Inc,
		planetv1connect.ClickServiceClickProcedure,
	), nil
}
