package internal_fill_charges_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/fill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/internal_fill_charges_handler"
)

const (
	grant   = "5a4f0e43-1d3b-5c2a-9c41-6f2c7e1d8b90"
	account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
)

type filling struct {
	filled []fill_usecase.In
}

func (f *filling) Execute(_ context.Context, in fill_usecase.In) error {
	f.filled = append(f.filled, in)
	return nil
}

func request() *planetv1.FillChargesRequest {
	return &planetv1.FillChargesRequest{
		GrantId: grant, AccountId: account,
		Kinds: []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_SHIELDS, planetv1.BonusKind_BONUS_KIND_BOMB},
	}
}

func TestAFillIsMadeAtOnceInTheWalletsWords(t *testing.T) {
	useCase := &filling{}

	_, err := internal_fill_charges_handler.New(useCase).FillCharges(t.Context(), connect.NewRequest(request()))

	require.NoError(t, err)
	require.Len(t, useCase.filled, 1)
	assert.Equal(t, grant, useCase.filled[0].Grant.String())
	assert.Equal(t, charges.Holder(account), useCase.filled[0].Holder)
	assert.Equal(t, []charges.Kind{charges.KindShields, charges.KindBomb}, useCase.filled[0].Kinds)
}

func TestABadFillIsRefusedAndFillsNothing(t *testing.T) {
	for name, breaks := range map[string]func(*planetv1.FillChargesRequest){
		"no grant id": func(r *planetv1.FillChargesRequest) { r.GrantId = "" },
		"no account":  func(r *planetv1.FillChargesRequest) { r.AccountId = "a-guest" },
		"no kind":     func(r *planetv1.FillChargesRequest) { r.Kinds = nil },
		"an unknown kind": func(r *planetv1.FillChargesRequest) {
			r.Kinds = append(r.Kinds, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED)
		},
	} {
		t.Run(name, func(t *testing.T) {
			useCase := &filling{}
			bad := request()
			breaks(bad)

			_, err := internal_fill_charges_handler.New(useCase).FillCharges(t.Context(), connect.NewRequest(bad))

			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.Empty(t, useCase.filled)
		})
	}
}
