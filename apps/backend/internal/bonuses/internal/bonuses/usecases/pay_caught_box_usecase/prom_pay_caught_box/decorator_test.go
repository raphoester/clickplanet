package prom_pay_caught_box_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase/prom_pay_caught_box"
)

type stubUseCase struct {
	out pay_caught_box_usecase.Out
	err error
}

func (s *stubUseCase) Execute(context.Context, bonuses.CaughtBox) (pay_caught_box_usecase.Out, error) {
	return s.out, s.err
}

func TestABoxThatPaysNothingIsCountedApartFromOnePaidAndOneFailed(t *testing.T) {
	registry := prometheus.NewRegistry()
	useCase := &stubUseCase{}
	decorator := prom_pay_caught_box.New(useCase, registry)

	for _, each := range []stubUseCase{
		{out: pay_caught_box_usecase.Out{Paid: true}},
		{out: pay_caught_box_usecase.Out{Paid: false}},
		{err: errors.New("planet is gone")},
		{out: pay_caught_box_usecase.Out{Paid: true}},
	} {
		*useCase = each
		_, _ = decorator.Execute(t.Context(), bonuses.CaughtBox{})
	}

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP bonus_box_rewards_total Boxes caught the bonuses module drew a reward for, by outcome: paid, nothing when the hand had room for no kind weighed, or failed
# TYPE bonus_box_rewards_total counter
bonus_box_rewards_total{outcome="failed"} 1
bonus_box_rewards_total{outcome="nothing"} 1
bonus_box_rewards_total{outcome="paid"} 2
`), "bonus_box_rewards_total"))
}
