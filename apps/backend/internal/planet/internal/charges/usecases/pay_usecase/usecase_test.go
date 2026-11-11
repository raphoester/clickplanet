package pay_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice = charges.Holder("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

var at = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

func setUp() (*pay_usecase.UseCase, *charges.MemoryStore) {
	hands := charges.NewMemoryStore()
	rules := charges.TestRules()
	rules.Spread.Clicks = 8
	wallet := charges.NewCharges(hands, func() charges.Rules { return rules })
	return pay_usecase.New(cpatomic.FakeTransactioner{}, inmemory_grant_store.New(time.Minute), wallet,
		cptime.NewFixedClock(at)), hands
}

func TestAPaymentIsGrantedAndAnswersWhatWasKeptAndTheHandAfter(t *testing.T) {
	useCase, hands := setUp()
	hands.Keep(alice, charges.Held{SpreadClicks: 7, Bomb: true})

	out, err := useCase.Execute(t.Context(), pay_usecase.In{
		Grant: charges.GrantID{15: 1}, Holder: alice, Kind: charges.KindSpreadClicks, Amount: 4,
	})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Kept, "the pool holds 8, and 7 were in it")
	assert.Equal(t, charges.Held{SpreadClicks: 8, Bomb: true}, out.Held)
	assert.Equal(t, out.Held, hands.Kept(alice))
}

func TestAPaymentToldAgainKeepsNothingAndAnswersTheHandAsItIs(t *testing.T) {
	useCase, hands := setUp()
	in := pay_usecase.In{Grant: charges.GrantID{15: 1}, Holder: alice, Kind: charges.KindShields, Amount: 5}

	_, err := useCase.Execute(t.Context(), in)
	require.NoError(t, err)
	again, err := useCase.Execute(t.Context(), in)
	require.NoError(t, err)

	assert.Zero(t, again.Kept)
	assert.Equal(t, charges.Held{Shields: 5}, again.Held)
	assert.Equal(t, charges.Held{Shields: 5}, hands.Kept(alice), "granted once")
}

func TestEachGrantIsPaidOnItsOwn(t *testing.T) {
	useCase, hands := setUp()

	for i := range 2 {
		_, err := useCase.Execute(t.Context(), pay_usecase.In{
			Grant: charges.GrantID{15: byte(i + 1)}, Holder: alice, Kind: charges.KindShields, Amount: 5,
		})
		require.NoError(t, err)
	}

	assert.Equal(t, charges.Held{Shields: 10}, hands.Kept(alice))
}

func TestACallerWithNoAccountIsPaidNothing(t *testing.T) {
	useCase, _ := setUp()

	out, err := useCase.Execute(t.Context(), pay_usecase.In{
		Grant: charges.GrantID{15: 1}, Holder: charges.NoHolder, Kind: charges.KindBomb, Amount: 1,
	})
	require.NoError(t, err)

	assert.Equal(t, pay_usecase.Out{}, out)
}

type failingGrants struct{}

func (failingGrants) Record(context.Context, charges.GrantID, time.Time) (bool, error) {
	return false, errors.New("postgres is down")
}

func TestAGrantThatCannotBeKeptIsNotPaid(t *testing.T) {
	hands := charges.NewMemoryStore()
	useCase := pay_usecase.New(cpatomic.FakeTransactioner{}, failingGrants{},
		charges.NewCharges(hands, charges.TestRules), cptime.NewFixedClock(at))

	_, err := useCase.Execute(t.Context(), pay_usecase.In{
		Grant: charges.GrantID{15: 1}, Holder: alice, Kind: charges.KindBomb, Amount: 1,
	})

	require.Error(t, err)
	assert.Equal(t, charges.Held{}, hands.Kept(alice))
}
