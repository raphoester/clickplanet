package prom_claim_box_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase/prom_claim_box"
)

type stubUseCase struct{ err error }

func (s *stubUseCase) Execute(context.Context, claim_box_usecase.In) error {
	return s.err
}

func TestAFailedTellIsCountedApartFromARefusal(t *testing.T) {
	registry := prometheus.NewRegistry()
	useCase := &stubUseCase{}
	decorator, _ := prom_claim_box.New(useCase, registry)

	down := errors.New("postgres is down")
	for _, err := range []error{nil, claim_box_usecase.ErrNoSuchBox, down, down} {
		useCase.err = err
		_ = decorator.Execute(t.Context(), claim_box_usecase.In{})
	}

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP bonus_claims_total Bonus boxes claimed, by outcome
# TYPE bonus_claims_total counter
bonus_claims_total{outcome="failed"} 2
bonus_claims_total{outcome="granted"} 1
bonus_claims_total{outcome="refused"} 1
`), "bonus_claims_total"))
}
