package prom_claim_bonus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase/prom_claim_bonus"
)

type stubUseCase struct{ err error }

func (s *stubUseCase) Execute(context.Context, claim_bonus_usecase.In) (claim_bonus_usecase.Out, error) {
	return claim_bonus_usecase.Out{}, s.err
}

func TestAFailedGrantIsCountedApartFromARefusal(t *testing.T) {
	registry := prometheus.NewRegistry()
	useCase := &stubUseCase{}
	decorator, _ := prom_claim_bonus.New(useCase, registry)

	down := errors.New("postgres is down")
	for _, err := range []error{nil, claim_bonus_usecase.ErrNoSuchBonus, down, down} {
		useCase.err = err
		_, _ = decorator.Execute(t.Context(), claim_bonus_usecase.In{})
	}

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP bonus_claims_total Bonus boxes claimed, by outcome
# TYPE bonus_claims_total counter
bonus_claims_total{outcome="failed"} 2
bonus_claims_total{outcome="granted"} 1
bonus_claims_total{outcome="refused"} 1
`), "bonus_claims_total"))
}
