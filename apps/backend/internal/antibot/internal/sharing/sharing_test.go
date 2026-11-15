package sharing_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sharing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const home = "203.0.113.7"

func production() sharing.Rules {
	return sharing.Rules{Enabled: true, Window: time.Hour, MaxAccounts: 3}
}

type harness struct {
	tracker *sharing.Tracker
	clock   *cptime.FixedClock
	rules   sharing.Rules
}

func newHarness() *harness {
	h := &harness{clock: cptime.NewFixedClock(time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)), rules: production()}
	h.tracker = sharing.New(func() sharing.Rules { return h.rules }, h.clock)
	return h
}

// Each try comes a minute after the one before.
func (h *harness) try(scope, account string) (string, bool) {
	h.clock.Advance(time.Minute)
	return h.tracker.Tried(detect.Click{Scope: scope, Account: account, At: h.clock.Now()})
}

func (h *harness) tries(scope, account string, n int) {
	for range n {
		_, _ = h.try(scope, account)
	}
}

func TestAGuestPlayedBetweenTwoTriesOfAnAccountIsFoundOnTheAccountsNextTry(t *testing.T) {
	h := newHarness()

	h.tries(home, "main", 3)
	_, found := h.try(home, "guest")
	require.False(t, found, "the guest only followed the account so far")

	alongside, found := h.try(home, "main")
	require.True(t, found)
	assert.Equal(t, "guest", alongside)

	alongside, found = h.try(home, "guest")
	require.True(t, found, "the account tried between two tries of the guest")
	assert.Equal(t, "main", alongside)
}

func TestAGuestThatSignsInToItsAccountPlaysInTurnsWithNobody(t *testing.T) {
	h := newHarness()

	h.tries(home, "guest", 10)
	for range 10 {
		_, found := h.try(home, "account")
		assert.False(t, found)
	}
}

func TestEachSpanIsAnsweredOnce(t *testing.T) {
	h := newHarness()

	h.tries(home, "main", 2)
	h.tries(home, "alt", 2)
	_, found := h.try(home, "main")
	require.True(t, found)

	h.tries(home, "alt", 1)
	_, found = h.try(home, "main")
	assert.False(t, found)
}

func TestTriesOutsideTheWindowDoNotCount(t *testing.T) {
	h := newHarness()

	h.tries(home, "parent", 2)
	h.tries(home, "child", 2)
	h.clock.Advance(2 * time.Hour)

	_, found := h.try(home, "parent")
	assert.False(t, found, "the child's tries lapsed, and so did the parent's first ones")
}

func TestTwoScopesNeverMeet(t *testing.T) {
	h := newHarness()

	h.tries(home, "main", 2)
	h.tries("198.51.100.0/64", "alt", 2)
	_, found := h.try(home, "main")
	assert.False(t, found)
}

func TestAScopeWithMoreAccountsThanTheBoundIsASharedNetworkAndNobodyIsFound(t *testing.T) {
	h := newHarness()

	for _, pupil := range []string{"ada", "grace", "alan", "barbara"} {
		h.tries(home, pupil, 1)
	}
	_, found := h.try(home, "ada")
	assert.False(t, found, "four accounts in the hour, over the bound of three")

	h.rules.MaxAccounts = 4
	alongside, found := h.try(home, "ada")
	require.True(t, found)
	assert.Equal(t, "barbara", alongside, "the one that tried last")
}

func TestOffNobodyIsFoundAndTheTriesStillCountOnceOn(t *testing.T) {
	h := newHarness()
	h.rules.Enabled = false

	h.tries(home, "main", 2)
	h.tries(home, "alt", 2)
	_, found := h.try(home, "main")
	require.False(t, found)

	h.rules.Enabled = true
	alongside, found := h.try(home, "main")
	require.True(t, found)
	assert.Equal(t, "alt", alongside)
}

func TestAClickWithNoAccountOrScopeIsNotKept(t *testing.T) {
	h := newHarness()

	h.tries(home, "main", 1)
	_, _ = h.tracker.Tried(detect.Click{Scope: home, At: h.clock.Now().Add(time.Second)})
	_, found := h.try(home, "main")
	assert.False(t, found)
}

func TestTheRulesRefuseAWindowOfZeroAndABoundUnderAPair(t *testing.T) {
	require.NoError(t, production().Validate())

	rules := production()
	rules.Window, rules.MaxAccounts = 0, 1
	err := rules.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window is 0s, and must be above 0")
	assert.Contains(t, err.Error(), "maxAccounts is 1, under the two accounts a pair is (2)")
}
