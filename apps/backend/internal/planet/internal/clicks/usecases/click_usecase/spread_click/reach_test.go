package spread_click_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/reach_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
)

type line map[uint32][]uint32

func (l line) Neighbours(id uint32) []uint32 { return l[id] }

var row = line{1: {2}, 2: {1, 3}, 3: {2, 4}, 4: {3, 5}, 5: {4}}

type field struct {
	mu     sync.Mutex
	owners map[uint32]string
}

func (f *field) HoldsAny(flag string, tiles []uint32) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, tile := range tiles {
		if f.owners[tile] == flag {
			return true
		}
	}
	return false
}

func (f *field) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.owners[in.TileID] = in.CountryID
	return click_usecase.Out{Outcome: clicks.Taken}, nil
}

func (f *field) Spread(_ context.Context, _ uint32, flag string, tiles []uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, tile := range tiles {
		f.owners[tile] = flag
	}
	return nil
}

func (f *field) Owners() map[uint32]string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.owners
}

type noBridges struct{}

func (noBridges) Across(uint32) []uint32 { return nil }

type holdFirst struct {
	inner   click_usecase.IUseCase
	refuse  error
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (h *holdFirst) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if h.calls.Add(1) == 1 {
		close(h.entered)
		<-h.release
		if h.refuse != nil {
			return click_usecase.Out{}, h.refuse
		}
	}
	return h.inner.Execute(ctx, in)
}

type chain struct {
	clicks click_usecase.IUseCase
	field  *field
	hands  *charges.MemoryStore
	gate   *holdFirst
}

func reachingChain(t *testing.T, spreadClicks int, rules func() charges.Rules, refuse error) *chain {
	t.Helper()

	c := &chain{field: &field{owners: map[uint32]string{1: "fr"}}, hands: charges.NewMemoryStore()}
	c.hands.Keep(caller, charges.Held{SpreadClicks: spreadClicks})
	reach := clicks.NewReach(func() clicks.Reaching { return true }, row, noBridges{}, c.field)
	c.gate = &holdFirst{inner: c.field, refuse: refuse, entered: make(chan struct{}), release: make(chan struct{})}
	c.clicks = spread_click.New(reach_click.New(c.gate, reach), charges.NewCharges(c.hands, rules), reach, c.field,
		&recordingPublisher{})

	return c
}

func TestWithTheReachASpreadingClickTakesACentreTwoTilesOutAndAPlainOneCannot(t *testing.T) {
	c := reachingChain(t, 1, charges.TestRules, nil)
	close(c.gate.release)

	_, plain := c.clicks.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr"})
	_, spread := c.clicks.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr", Spread: true})

	require.ErrorIs(t, plain, clicks.ErrOutOfReach)
	require.NoError(t, spread)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr"}, c.field.Owners(),
		"the spread fills the gap, and the front moves three tiles deep")
	assert.Zero(t, c.hands.Kept(caller).SpreadClicks)
}

func TestASpreadingClickOutOfEvenItsDiscsReachGivesItsSpreadClickBack(t *testing.T) {
	c := reachingChain(t, 1, charges.TestRules, nil)
	close(c.gate.release)

	_, err := c.clicks.Execute(played(t), click_usecase.In{TileID: 4, CountryID: "fr", Spread: true})

	require.ErrorIs(t, err, clicks.ErrOutOfReach)
	assert.Equal(t, 1, c.hands.Kept(caller).SpreadClicks)
	assert.Equal(t, map[uint32]string{1: "fr"}, c.field.Owners())
}

func TestTwoClicksRacingForTheLastSpreadClickLandOutsideThePlainReachOnlyWithTheSpread(t *testing.T) {
	c := reachingChain(t, 1, charges.TestRules, nil)
	first := make(chan error)
	go func() {
		_, err := c.clicks.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr", Spread: true})
		first <- err
	}()
	<-c.gate.entered

	_, second := c.clicks.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr", Spread: true})
	close(c.gate.release)

	require.ErrorIs(t, second, clicks.ErrOutOfReach, "the last spread click is the first one's: this is a plain click")
	require.NoError(t, <-first)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr"}, c.field.Owners())
	assert.Zero(t, c.hands.Kept(caller).SpreadClicks)
}

func TestTheLastSpreadClickOfAClickRefusedInsideComesBack(t *testing.T) {
	throttled := errors.New("throttled")
	c := reachingChain(t, 1, charges.TestRules, throttled)
	first := make(chan error)
	go func() {
		_, err := c.clicks.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr", Spread: true})
		first <- err
	}()
	<-c.gate.entered

	_, second := c.clicks.Execute(played(t), click_usecase.In{TileID: 3, CountryID: "fr", Spread: true})
	close(c.gate.release)

	require.ErrorIs(t, second, clicks.ErrOutOfReach)
	require.ErrorIs(t, <-first, throttled)
	assert.Equal(t, map[uint32]string{1: "fr"}, c.field.Owners())
	assert.Equal(t, 1, c.hands.Kept(caller).SpreadClicks, "refused, it spread nothing and spent nothing")
}

func TestUnlimitedSpreadsWidenEveryClicksReach(t *testing.T) {
	c := reachingChain(t, 0, spreading(2, true), nil)
	close(c.gate.release)

	_, err := c.clicks.Execute(played(t), click_usecase.In{TileID: 4, CountryID: "fr", Spread: true})

	require.NoError(t, err)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr", 5: "fr"}, c.field.Owners())
}
