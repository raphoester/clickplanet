package claim_bonus_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubRegistry struct {
	reward    bonuses.Reward
	claimable bool

	token     string
	entrant   bonuses.Entrant
	scope     string
	published []bonuses.Taken
}

func (s *stubRegistry) Claim(token string, entrant bonuses.Entrant, scope string) (bonuses.Reward, bool) {
	s.token, s.entrant, s.scope = token, entrant, scope
	if !s.claimable {
		return bonuses.Reward{}, false
	}

	return s.reward, true
}

func (s *stubRegistry) Publish(taken bonuses.Taken) { s.published = append(s.published, taken) }

const guest bonuses.Holder = "a-guest"

func charges() (bonuses.Charges, *bonuses.MemoryStore) {
	store := bonuses.NewMemoryStore()
	return bonuses.NewCharges(store, bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3, Shields: 12}), store
}

func granting(kind bonuses.Kind) *stubRegistry {
	return &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: kind, Amount: 1}}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), string(guest))
}

func TestAClaimHandsTheChargeToTheAccount(t *testing.T) {
	registry := granting(bonuses.KindRefill)
	charger, store := charges()

	out, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, "a-token", registry.token)
	assert.Equal(t, bonuses.Entrant("1.2.3.4"), registry.entrant, "a guest's offer is the scope's")
	assert.Equal(t, bonuses.Held{Refill: true}, store.Kept(guest))
	assert.Equal(t, bonuses.KindRefill, out.Kind)
	assert.Equal(t, bonuses.Held{Refill: true}, out.Held)
}

func TestALinkedAccountClaimsWhatWasOfferedToTheAccount(t *testing.T) {
	registry := granting(bonuses.KindRefill)
	charger, _ := charges()
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-player"))

	_, err := claim_bonus_usecase.New(registry, charger).
		Execute(ctx, claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Entrant("account:a-player"), registry.entrant)
	assert.Equal(t, "1.2.3.4", registry.scope)
}

func TestEveryKindIsGrantedAsACharge(t *testing.T) {
	for _, kind := range bonuses.Kinds {
		charger, store := charges()

		_, err := claim_bonus_usecase.New(granting(kind), charger).
			Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
		require.NoError(t, err)

		assert.Equal(t, 1, store.Kept(guest).Count(kind), kind)
	}
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	registry := granting(bonuses.KindBomb)
	charger, _ := charges()

	_, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "jp"})
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Taken{{CountryID: "jp", Kind: bonuses.KindBomb}}, registry.published)
}

func TestARefusedClaimGrantsNothingAndAnnouncesNothing(t *testing.T) {
	registry := &stubRegistry{claimable: false}
	charger, store := charges()

	_, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "not-mine"})

	require.ErrorIs(t, err, claim_bonus_usecase.ErrNoSuchBonus)
	assert.Equal(t, bonuses.Held{}, store.Kept(guest))
	assert.Empty(t, registry.published)
}

func TestTheAmountTheBoxGaveIsGrantedAndAnswered(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 3}}
	charger, store := charges()

	out, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, store.Kept(guest))
	assert.Equal(t, 3, out.Amount)
	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, out.Held)
}

func TestTheAnswerSaysOnlyWhatThePoolKept(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 4}}
	charger, _ := charges()
	require.NoError(t, charger.Grant(t.Context(), guest, bonuses.KindSpreadClicks, 7))

	out, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Amount, "the pool held 7 of 8, so a box of 4 added 1")
	assert.Equal(t, bonuses.Held{SpreadClicks: 8}, out.Held)
}

func TestAStoreThatFailsFailsTheClaimAndAnnouncesNothing(t *testing.T) {
	registry := granting(bonuses.KindBomb)
	charger, store := charges()
	down := errors.New("postgres is down")
	store.FailWith(down)

	_, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "jp"})

	require.ErrorIs(t, err, down)
	assert.Empty(t, registry.published, "a catch is announced once the charge is held")
}
