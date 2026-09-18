package claim_bonus_usecase_test

import (
	"context"
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
	scope     string
	published []bonuses.Taken
}

func (s *stubRegistry) Claim(token string, scope string) (bonuses.Reward, bool) {
	s.token, s.scope = token, scope
	if !s.claimable {
		return bonuses.Reward{}, false
	}

	return s.reward, true
}

func (s *stubRegistry) Publish(taken bonuses.Taken) { s.published = append(s.published, taken) }

// stubCharger records the charges granted, and answers what they add up to.
type stubCharger struct {
	granted []grantedCharge
}

type grantedCharge struct {
	holder bonuses.Holder
	kind   bonuses.Kind
}

func (s *stubCharger) Grant(holder bonuses.Holder, kind bonuses.Kind) {
	s.granted = append(s.granted, grantedCharge{holder: holder, kind: kind})
}

func (s *stubCharger) Held(holder bonuses.Holder) bonuses.Held {
	var held bonuses.Held
	for _, g := range s.granted {
		if g.holder != holder {
			continue
		}
		switch g.kind {
		case bonuses.KindRefill:
			held.Refill = true
		case bonuses.KindBomb:
			held.Bomb = true
		case bonuses.KindEncloseClicks:
			held.Enclose = true
		case bonuses.KindSpreadClicks:
			held.SpreadClicks = 8
		}
	}

	return held
}

func granting(kind bonuses.Kind) *stubRegistry {
	return &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: kind}}
}

// played is the test's context, from an address, with an account on it.
func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

func TestAClaimHandsTheChargeToTheAccount(t *testing.T) {
	registry, charger := granting(bonuses.KindRefill), &stubCharger{}

	out, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, "a-token", registry.token)
	assert.Equal(t, "1.2.3.4", registry.scope, "the offer is the scope's")
	assert.Equal(t, []grantedCharge{{holder: "a-guest", kind: bonuses.KindRefill}}, charger.granted)
	assert.Equal(t, bonuses.KindRefill, out.Kind)
	assert.Equal(t, bonuses.Held{Refill: true}, out.Held)
}

func TestEveryKindIsGrantedAsACharge(t *testing.T) {
	for _, kind := range bonuses.Kinds {
		charger := &stubCharger{}

		_, err := claim_bonus_usecase.New(granting(kind), charger).
			Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
		require.NoError(t, err)

		assert.Equal(t, []grantedCharge{{holder: "a-guest", kind: kind}}, charger.granted)
	}
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	registry := granting(bonuses.KindBomb)

	_, err := claim_bonus_usecase.New(registry, &stubCharger{}).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "jp"})
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Taken{{CountryID: "jp", Kind: bonuses.KindBomb}}, registry.published)
}

func TestARefusedClaimGrantsNothingAndAnnouncesNothing(t *testing.T) {
	registry, charger := &stubRegistry{claimable: false}, &stubCharger{}

	_, err := claim_bonus_usecase.New(registry, charger).
		Execute(played(t), claim_bonus_usecase.In{Token: "not-mine"})

	require.ErrorIs(t, err, claim_bonus_usecase.ErrNoSuchBonus)
	assert.Empty(t, charger.granted)
	assert.Empty(t, registry.published)
}
