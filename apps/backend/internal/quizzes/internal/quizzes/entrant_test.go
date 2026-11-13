package quizzes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

const sharedAddress = "1.2.3.4"

var (
	alice = Caller{Scope: sharedAddress, Account: "alice", Linked: true}
	bob   = Caller{Scope: sharedAddress, Account: "bob", Linked: true}
	guest = Caller{Scope: sharedAddress, Account: "a-guest"}
)

func playingAs(t *testing.T, r *Registry, caller Caller) <-chan Event {
	t.Helper()

	events := attend(t, r, EntrantOf(caller))
	r.Clicked(Click{Caller: caller})

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
	assert.Equal(t, Entrant(sharedAddress), EntrantOf(Caller{Scope: sharedAddress}))
}

func TestTwoLinkedAccountsBehindOneAddressAreEachAskedTheirOwnQuiz(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)
	hers, his := playingAs(t, registry, alice), playingAs(t, registry, bob)

	waitOutQuiz(t, registry, clock)

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
	assert.Equal(t, ratings.Right, herAnswer.Outcome)
	assert.Equal(t, ratings.Right, hisAnswer.Outcome)
}

func TestALinkedAccountsMissIsReportedByItsNetwork(t *testing.T) {
	registry, clock := newQuizzingRegistry(t)

	var missedBy []string
	registry.Observe(Report{Lapsed: func(scope string) { missedBy = append(missedBy, scope) }})

	events := playingAs(t, registry, alice)
	waitOutQuiz(t, registry, clock)
	require.NotNil(t, quizOffered(t, events))

	clock.Advance(bannerFor + time.Second)
	registry.sweep()

	assert.Equal(t, []string{sharedAddress}, missedBy)
}
