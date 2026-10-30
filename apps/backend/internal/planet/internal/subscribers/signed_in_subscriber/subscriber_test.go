package signed_in_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_sign_in_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/signed_in_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type grants struct {
	recorded *cpcolls.Set[bonuses.GrantID]
}

func (g grants) Record(_ context.Context, grant bonuses.GrantID, _ time.Time) (bool, error) {
	if g.recorded.Contains(grant) {
		return false, nil
	}
	g.recorded.Add(grant)
	return true, nil
}

const (
	guest = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	other = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
)

func setUp() (signed_in_subscriber.Subscriber, bonuses.Charges) {
	rules := bonuses.TestRules()
	rules.SignInRefill = true
	inForce := func() bonuses.Rules { return rules }
	charges := bonuses.NewCharges(bonuses.NewMemoryStore(), inForce)
	useCase := grant_sign_in_refill_usecase.New(cpatomic.FakeTransactioner{}, grants{recorded: cpcolls.NewSet[bonuses.GrantID]()},
		charges, inForce, cptime.NewFixedClock(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)))
	return signed_in_subscriber.New(useCase), charges
}

func refillHeld(t *testing.T, charges bonuses.Charges, account string) bool {
	t.Helper()

	held, err := charges.Held(t.Context(), bonuses.Holder(account))
	require.NoError(t, err)
	return held.Refill
}

func TestAGuestThatLinksItsFirstIdentityEarnsARefill(t *testing.T) {
	subscriber, charges := setUp()

	require.NoError(t, subscriber.Handle(t.Context(), &authv1.SignedIn{PreviousAccountId: guest, AccountId: guest, GuestLinked: true}))

	assert.True(t, refillHeld(t, charges, guest))
}

func TestEveryOtherSignInEarnsNothing(t *testing.T) {
	for name, event := range map[string]*authv1.SignedIn{
		"a second provider":        {PreviousAccountId: guest, AccountId: guest},
		"into an account that was": {PreviousAccountId: guest, AccountId: other},
		"a new account":            {AccountId: other},
		"back in after a sign-out": {PreviousAccountId: other, AccountId: guest},
	} {
		t.Run(name, func(t *testing.T) {
			subscriber, charges := setUp()

			require.NoError(t, subscriber.Handle(t.Context(), event))

			assert.False(t, refillHeld(t, charges, guest))
			assert.False(t, refillHeld(t, charges, other))
		})
	}
}

func TestAnEventThatNamesNoAccountIsRefused(t *testing.T) {
	subscriber, _ := setUp()

	err := subscriber.Handle(t.Context(), &authv1.SignedIn{AccountId: "nope", GuestLinked: true})

	require.ErrorIs(t, err, bonuses.ErrInvalidHolder)
}
