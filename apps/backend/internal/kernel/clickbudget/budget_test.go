package clickbudget

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

const retention = time.Hour

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestBudget(clicks int) (*Budget, *fakeClock) {
	clock := &fakeClock{now: epoch}
	return New(Config{Clicks: clicks}, retention, clock), clock
}

func TestASessionSpendsExactlyItsBudget(t *testing.T) {
	budget, _ := newTestBudget(3)

	for i := range 3 {
		require.True(t, budget.Spend("session-a"), "click %d should be affordable", i+1)
	}

	require.False(t, budget.Spend("session-a"), "the fourth click is past the budget")
}

func TestAnExhaustedSessionStaysExhausted(t *testing.T) {
	budget, clock := newTestBudget(1)

	require.True(t, budget.Spend("session-a"))
	require.False(t, budget.Spend("session-a"))

	// The whole point of not being a token bucket: waiting does not buy more.
	// The caller is meant to mint another session, not to sit it out.
	clock.advance(retention / 2)
	require.False(t, budget.Spend("session-a"))
}

func TestSessionsDoNotShareABudget(t *testing.T) {
	budget, _ := newTestBudget(1)

	require.True(t, budget.Spend("session-a"))
	require.False(t, budget.Spend("session-a"))

	require.True(t, budget.Spend("session-b"), "another session has its own budget")
}

func TestATallyIsKeptForAsLongAsItsTokenCouldBeValid(t *testing.T) {
	budget, clock := newTestBudget(1)

	require.True(t, budget.Spend("session-a"))

	// Forgetting a tally early is the one bug that matters here: the token is
	// still inside its TTL, so the caller would be handed a second budget for
	// the same attestation.
	clock.advance(retention - time.Second)
	budget.sweep()

	require.False(t, budget.Spend("session-a"), "the tally must survive as long as the token does")
}

func TestATallyIsForgottenOnceItsTokenCannotBe(t *testing.T) {
	budget, clock := newTestBudget(1)

	require.True(t, budget.Spend("session-a"))

	// At retention the token is expired, so Verify refuses it upstream and the
	// tally is only costing memory.
	clock.advance(retention)
	budget.sweep()

	require.Empty(t, budget.tallies, "an expired session's tally should be swept")
}

func TestNoBudgetIsConfiguredWhenClicksIsNotPositive(t *testing.T) {
	for _, clicks := range []int{0, -1} {
		require.Nil(t, New(Config{Clicks: clicks}, retention, nil))
	}
}

func TestANilBudgetSpendsFreely(t *testing.T) {
	// How a deployment that has not sized a budget leaves the click path alone.
	var budget *Budget

	for range 100 {
		require.True(t, budget.Spend("session-a"))
	}
}

func TestConcurrentSpendsHandOutTheBudgetExactlyOnce(t *testing.T) {
	const clicks = 50
	budget, _ := newTestBudget(clicks)

	var wg sync.WaitGroup
	granted := make([]bool, 200)

	for i := range granted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			granted[i] = budget.Spend("session-a")
		}()
	}
	wg.Wait()

	allowed := 0
	for _, ok := range granted {
		if ok {
			allowed++
		}
	}

	require.Equal(t, clicks, allowed, "the budget must not overspend under concurrency")
}
