package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const sharedAddress = "1.2.3.4"

var (
	alice = clicks.Payer{Scope: sharedAddress, Account: "alice", Linked: true}
	bob   = clicks.Payer{Scope: sharedAddress, Account: "bob", Linked: true}
	guest = clicks.Payer{Scope: sharedAddress, Account: "a-guest"}
)

func playingAs(t *testing.T, r *Registry, payer clicks.Payer) <-chan Event {
	t.Helper()

	events := attend(t, r, EntrantOf(payer))
	r.Clicked(EntrantOf(payer), payer.Scope, HolderOf(payer))

	return events
}

func TestALinkedAccountIsItsOwnEntrantOnAnyNetwork(t *testing.T) {
	elsewhere := alice
	elsewhere.Scope = "5.6.7.8"

	assert.Equal(t, Entrant("account:alice"), EntrantOf(alice))
	assert.Equal(t, EntrantOf(alice), EntrantOf(elsewhere))
	assert.NotEqual(t, EntrantOf(alice), EntrantOf(bob))
}

func TestAGuestOrACallerWithNoAccountIsItsNetwork(t *testing.T) {
	assert.Equal(t, Entrant(sharedAddress), EntrantOf(guest))
	assert.Equal(t, Entrant(sharedAddress), EntrantOf(clicks.Payer{Scope: sharedAddress}))
}

func TestTwoLinkedAccountsBehindOneAddressAreEachOfferedAndClaimTheirOwnBox(t *testing.T) {
	registry, clock := newTestRegistry()
	hers, his := playingAs(t, registry, alice), playingAs(t, registry, bob)

	waitOut(registry, clock)

	herOffer, hisOffer := offered(t, hers), offered(t, his)
	require.NotNil(t, herOffer)
	require.NotNil(t, hisOffer)
	assert.NotEqual(t, herOffer.Token, hisOffer.Token)

	_, stolen := registry.Claim(herOffer.Token, EntrantOf(bob), sharedAddress)
	assert.False(t, stolen)

	_, herClaim := registry.Claim(herOffer.Token, EntrantOf(alice), sharedAddress)
	_, hisClaim := registry.Claim(hisOffer.Token, EntrantOf(bob), sharedAddress)
	assert.True(t, herClaim)
	assert.True(t, hisClaim)
}

func TestAGuestBehindTheAddressOfALinkedAccountIsOnTheAddressSchedule(t *testing.T) {
	registry, clock := newTestRegistry()
	hers, theirs := playingAs(t, registry, alice), playingAs(t, registry, guest)

	waitOut(registry, clock)

	herOffer, theirOffer := offered(t, hers), offered(t, theirs)
	require.NotNil(t, herOffer)
	require.NotNil(t, theirOffer)

	_, claimed := registry.Claim(theirOffer.Token, EntrantOf(guest), sharedAddress)
	assert.True(t, claimed)
	assert.Contains(t, registry.callers, Entrant(sharedAddress))
	assert.Contains(t, registry.callers, Entrant("account:alice"))
}

func TestALinkedAccountsCatchIsReportedToTheAntibotByItsNetwork(t *testing.T) {
	registry, clock := newTestRegistry()

	var caughtBy []string
	registry.Observe(Report{Caught: func(scope string, _ time.Duration) { caughtBy = append(caughtBy, scope) }})

	events := playingAs(t, registry, alice)
	waitOut(registry, clock)
	offer := offered(t, events)
	require.NotNil(t, offer)

	_, claimed := registry.Claim(offer.Token, EntrantOf(alice), sharedAddress)
	require.True(t, claimed)

	assert.Equal(t, []string{sharedAddress}, caughtBy)
}

func TestALinkedAccountsMissIsReportedToTheAntibotByItsNetwork(t *testing.T) {
	registry, clock := newTestRegistry()

	var missedBy []string
	registry.Observe(Report{Lapsed: func(scope string) { missedBy = append(missedBy, scope) }})

	events := playingAs(t, registry, alice)
	waitOut(registry, clock)
	require.NotNil(t, offered(t, events))

	clock.Advance(registry.config.OfferTTL + time.Second)
	registry.sweep(t.Context())

	assert.Equal(t, []string{sharedAddress}, missedBy)
}

func TestTwoLinkedAccountsBehindOneAddressAreEachAskedTheirOwnQuiz(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	hers, his := playingAs(t, registry, alice), playingAs(t, registry, bob)

	waitOutQuiz(registry, clock)

	herQuiz, hisQuiz := quizOffered(t, hers), quizOffered(t, his)
	require.NotNil(t, herQuiz)
	require.NotNil(t, hisQuiz)

	_, stolen := registry.OpenQuiz(herQuiz.Token, EntrantOf(bob))
	assert.False(t, stolen)

	herAsked, herOpened := registry.OpenQuiz(herQuiz.Token, EntrantOf(alice))
	hisAsked, hisOpened := registry.OpenQuiz(hisQuiz.Token, EntrantOf(bob))
	require.True(t, herOpened)
	require.True(t, hisOpened)

	herAnswer, _ := registry.AnswerQuiz(herQuiz.Token, EntrantOf(alice), indexOf(herAsked.Options, rightAnswer))
	hisAnswer, _ := registry.AnswerQuiz(hisQuiz.Token, EntrantOf(bob), indexOf(hisAsked.Options, rightAnswer))
	assert.True(t, herAnswer.Correct)
	assert.True(t, hisAnswer.Correct)
}
