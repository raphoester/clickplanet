package claim_bonus_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/claim_bonus_usecase"
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

var guest = bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest"}

func granting(kind bonuses.Kind) *stubRegistry {
	return &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: kind, Amount: 1}}
}

func claimOf(token, country string) claim_bonus_usecase.In {
	return claim_bonus_usecase.In{Caller: guest, Token: token, CountryID: country}
}

func TestAClaimHandsTheChargeToTheAccount(t *testing.T) {
	registry := granting(bonuses.KindRefill)
	wallet := bonuses.NewFakeWallet()

	out, err := claim_bonus_usecase.New(registry, wallet, cpredis.NewRecordedEvents(), clock).
		Execute(t.Context(), claimOf("a-token", "fr"))
	require.NoError(t, err)

	assert.Equal(t, "a-token", registry.token)
	assert.Equal(t, bonuses.Entrant("1.2.3.4"), registry.entrant, "a guest's offer is the scope's")
	assert.Equal(t, bonuses.Hand{Refill: true}, wallet.Kept(guest.Account))
	assert.Equal(t, bonuses.KindRefill, out.Kind)
	assert.Equal(t, bonuses.Hand{Refill: true}, out.Hand)
}

func TestALinkedAccountClaimsWhatWasOfferedToTheAccount(t *testing.T) {
	registry := granting(bonuses.KindRefill)
	wallet := bonuses.NewFakeWallet()
	player := bonuses.Caller{Scope: "1.2.3.4", Account: "a-player", Linked: true}

	_, err := claim_bonus_usecase.New(registry, wallet, cpredis.NewRecordedEvents(), clock).
		Execute(t.Context(), claim_bonus_usecase.In{Caller: player, Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Entrant("account:a-player"), registry.entrant)
	assert.Equal(t, "1.2.3.4", registry.scope)
}

func TestEveryKindIsGrantedAsACharge(t *testing.T) {
	for _, kind := range bonuses.Kinds {
		wallet := bonuses.NewFakeWallet()

		_, err := claim_bonus_usecase.New(granting(kind), wallet, cpredis.NewRecordedEvents(), clock).
			Execute(t.Context(), claimOf("a-token", ""))
		require.NoError(t, err)

		require.Len(t, wallet.Payments(), 1)
		assert.Equal(t, bonuses.Payment{
			Grant: bonuses.OfferGrantOf("a-token"), Account: guest.Account, Kind: kind, Amount: 1,
		}, wallet.Payments()[0], kind)
	}
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	registry := granting(bonuses.KindBomb)
	wallet := bonuses.NewFakeWallet()

	_, err := claim_bonus_usecase.New(registry, wallet, cpredis.NewRecordedEvents(), clock).
		Execute(t.Context(), claimOf("a-token", "jp"))
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Taken{{CountryID: "jp", Kind: bonuses.KindBomb}}, registry.published)
}

func TestACatchIsToldToTheOtherModulesWithTheAccountThatCaughtIt(t *testing.T) {
	wallet := bonuses.NewFakeWallet()
	events := cpredis.NewRecordedEvents()

	_, err := claim_bonus_usecase.New(granting(bonuses.KindBomb), wallet, events, clock).
		Execute(t.Context(), claimOf("a-token", "jp"))
	require.NoError(t, err)

	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&bonusesv1.BonusClaimed{AccountId: string(guest.Account), ClaimedAt: timestamppb.New(now)},
		events.Published()[0]))
}

func TestARefusedClaimGrantsNothingAndAnnouncesNothing(t *testing.T) {
	registry := &stubRegistry{claimable: false}
	wallet := bonuses.NewFakeWallet()
	events := cpredis.NewRecordedEvents()

	_, err := claim_bonus_usecase.New(registry, wallet, events, clock).
		Execute(t.Context(), claimOf("not-mine", ""))

	require.ErrorIs(t, err, claim_bonus_usecase.ErrNoSuchBonus)
	assert.Equal(t, bonuses.Hand{}, wallet.Kept(guest.Account))
	assert.Empty(t, registry.published)
	assert.Empty(t, events.Published())
}

func TestTheAmountTheBoxGaveIsGrantedAndAnswered(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 3}}
	wallet := bonuses.NewFakeWallet()

	out, err := claim_bonus_usecase.New(registry, wallet, cpredis.NewRecordedEvents(), clock).
		Execute(t.Context(), claimOf("a-token", ""))
	require.NoError(t, err)

	assert.Equal(t, bonuses.Hand{SpreadClicks: 3}, wallet.Kept(guest.Account))
	assert.Equal(t, 3, out.Amount)
	assert.Equal(t, bonuses.Hand{SpreadClicks: 3}, out.Hand)
}

func TestTheAnswerSaysOnlyWhatThePoolKept(t *testing.T) {
	registry := &stubRegistry{claimable: true, reward: bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 4}}
	wallet := bonuses.NewFakeWallet()
	wallet.Size(bonuses.KindSpreadClicks, 8)
	wallet.Keep(guest.Account, bonuses.Hand{SpreadClicks: 7})

	out, err := claim_bonus_usecase.New(registry, wallet, cpredis.NewRecordedEvents(), clock).
		Execute(t.Context(), claimOf("a-token", ""))
	require.NoError(t, err)

	assert.Equal(t, 1, out.Amount, "the pool held 7 of 8, so a box of 4 added 1")
	assert.Equal(t, bonuses.Hand{SpreadClicks: 8}, out.Hand)
}

func TestAClaimIsGrantedWhenTheCallerLeavesAfterIt(t *testing.T) {
	wallet := bonuses.NewFakeWallet()
	ctx, leave := context.WithCancel(t.Context())
	leave()

	_, err := claim_bonus_usecase.New(granting(bonuses.KindBomb), wallet, cpredis.NewRecordedEvents(), clock).
		Execute(ctx, claimOf("a-token", ""))
	require.NoError(t, err)

	assert.Equal(t, []error{nil}, wallet.Gone(), "the payment runs on a context the caller's leaving does not end")
	assert.Equal(t, bonuses.Hand{Bomb: true}, wallet.Kept(guest.Account))
}

func TestAFailedGrantIsNotARefusalAndIsToldToNobody(t *testing.T) {
	registry := granting(bonuses.KindBomb)
	wallet := bonuses.NewFakeWallet()
	wallet.FailWith(errors.New("postgres is down"))
	events := cpredis.NewRecordedEvents()

	_, err := claim_bonus_usecase.New(registry, wallet, events, clock).
		Execute(t.Context(), claimOf("a-token", ""))

	require.Error(t, err)
	assert.NotErrorIs(t, err, claim_bonus_usecase.ErrNoSuchBonus)
	assert.Empty(t, registry.published)
	assert.Empty(t, events.Published())
}
