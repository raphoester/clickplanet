package internal_grant_charges_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/internal_grant_charges_handler"
)

const (
	grant   = "5a4f0e43-1d3b-5c2a-9c41-6f2c7e1d8b90"
	account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
)

type payment struct {
	paid []pay_usecase.In
}

func (p *payment) Execute(_ context.Context, in pay_usecase.In) (pay_usecase.Out, error) {
	p.paid = append(p.paid, in)
	return pay_usecase.Out{Kept: 3, Held: charges.Held{Shields: 9}}, nil
}

func request(lasting bool) *planetv1.InternalGrantChargesRequest {
	return &planetv1.InternalGrantChargesRequest{
		GrantId: grant, AccountId: account, Kind: planetv1.BonusKind_BONUS_KIND_SHIELDS, Amount: 5, Lasting: lasting,
	}
}

func TestAGrantIsPaidAtOnceAndAnswersWhatWasKeptAndTheHand(t *testing.T) {
	brief, lasting := &payment{}, &payment{}

	res, err := internal_grant_charges_handler.New(brief, lasting).GrantCharges(t.Context(), connect.NewRequest(request(false)))

	require.NoError(t, err)
	assert.Equal(t, uint32(3), res.Msg.GetKept())
	assert.Equal(t, uint32(9), res.Msg.GetCharges().GetShields())
	require.Len(t, brief.paid, 1)
	assert.Equal(t, grant, brief.paid[0].Grant.String())
	assert.Equal(t, charges.Holder(account), brief.paid[0].Holder)
	assert.Equal(t, charges.KindShields, brief.paid[0].Kind)
	assert.Equal(t, 5, brief.paid[0].Amount)
	assert.Empty(t, lasting.paid, "an offer's id is known for a while, in memory")
}

func TestALastingGrantIsKnownForGood(t *testing.T) {
	brief, lasting := &payment{}, &payment{}

	_, err := internal_grant_charges_handler.New(brief, lasting).GrantCharges(t.Context(), connect.NewRequest(request(true)))

	require.NoError(t, err)
	assert.Len(t, lasting.paid, 1)
	assert.Empty(t, brief.paid)
}

func TestABadGrantIsRefusedAndPaysNothing(t *testing.T) {
	for name, breaks := range map[string]func(*planetv1.InternalGrantChargesRequest){
		"no grant id":     func(r *planetv1.InternalGrantChargesRequest) { r.GrantId = "" },
		"no account":      func(r *planetv1.InternalGrantChargesRequest) { r.AccountId = "a-guest" },
		"no kind":         func(r *planetv1.InternalGrantChargesRequest) { r.Kind = planetv1.BonusKind_BONUS_KIND_UNSPECIFIED },
		"nothing to give": func(r *planetv1.InternalGrantChargesRequest) { r.Amount = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			paid := &payment{}
			bad := request(false)
			breaks(bad)

			_, err := internal_grant_charges_handler.New(paid, paid).GrantCharges(t.Context(), connect.NewRequest(bad))

			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			assert.Empty(t, paid.paid)
		})
	}
}
