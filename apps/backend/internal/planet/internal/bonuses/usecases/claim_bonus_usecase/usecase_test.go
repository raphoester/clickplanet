package claim_bonus_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	clock = cptime.NewFixedClock(now)
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
	rules := bonuses.TestRules()
	rules.Shield.Held = 12
	return bonuses.NewCharges(store, func() bonuses.Rules { return rules }), store
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

	out, err := claim_bonus_usecase.New(registry, charger, cpredis.NewRecordedEvents(), clock).
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

	_, err := claim_bonus_usecase.New(registry, charger, cpredis.NewRecordedEvents(), clock).
		Execute(ctx, claim_bonus_usecase.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Entrant("account:a-player"), registry.entrant)
	assert.Equal(t, "1.2.3.4", registry.scope)
}

func TestEveryKindIsGrantedAsACharge(t *testing.T) {
	for _, kind := range bonuses.Kinds {
		charger, store := charges()

		_, err := claim_bonus_usecase.New(granting(kind), charger, cpredis.NewRecordedEvents(), clock).
			Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
		require.NoError(t, err)

		assert.Equal(t, 1, store.Kept(guest).Count(kind), kind)
	}
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	registry := granting(bonuses.KindBomb)
	charger, _ := charges()

	_, err := claim_bonus_usecase.New(registry, charger, cpredis.NewRecordedEvents(), clock).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "jp"})
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Taken{{CountryID: "jp", Kind: bonuses.KindBomb}}, registry.published)
}

func TestACatchIsToldToTheOtherModulesWithTheAccountThatCaughtIt(t *testing.T) {
	charger, _ := charges()
	events := cpredis.NewRecordedEvents()

	_, err := claim_bonus_usecase.New(granting(bonuses.KindBomb), charger, events, clock).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token", CountryID: "jp"})
	require.NoError(t, err)

	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&planetv1.BonusClaimed{AccountId: string(guest), ClaimedAt: timestamppb.New(now)},
		events.Published()[0]))
}

func TestARefusedClaimGrantsNothingAndAnnouncesNothing(t *testing.T) {
	registry := &stubRegistry{claimable: false}
	charger, store := charges()
	events := cpredis.NewRecordedEvents()

	_, err := claim_bonus_usecase.New(registry, charger, events, clock).
		Execute(played(t), claim_bonus_usecase.In{Token: "not-mine"})

	require.ErrorIs(t, err, claim_bonus_usecase.ErrNoSuchBonus)
	assert.Equal(t, bonuses.Held{}, store.Kept(guest))
	assert.Empty(t, registry.published)
	assert.Empty(t, events.Published())
}

func TestTheAmountTheBoxGaveIsGrantedAndAnswered(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 3}}
	charger, store := charges()

	out, err := claim_bonus_usecase.New(registry, charger, cpredis.NewRecordedEvents(), clock).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, store.Kept(guest))
	assert.Equal(t, 3, out.Amount)
	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, out.Held)
}

func TestTheAnswerSaysOnlyWhatThePoolKept(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 4}}
	charger, store := charges()
	store.Keep(guest, bonuses.Held{SpreadClicks: 7})

	out, err := claim_bonus_usecase.New(registry, charger, cpredis.NewRecordedEvents(), clock).
		Execute(played(t), claim_bonus_usecase.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Amount, "the pool held 7 of 8, so a box of 4 added 1")
	assert.Equal(t, bonuses.Held{SpreadClicks: 8}, out.Held)
}
