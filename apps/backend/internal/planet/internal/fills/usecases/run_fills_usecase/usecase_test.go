package run_fills_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/run_fills_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	ada  bonuses.Holder = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob  bonuses.Holder = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
	cleo bonuses.Holder = "9e1f2a3b-4c5d-4e6f-8a7b-0c1d2e3f4a5b"
)

var (
	at    = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	sizes = bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3, EnclosureMaxTiles: 25, Shields: 30}
)

func full(fill bonuses.FillID) bonuses.Held {
	return bonuses.Held{Refill: true, Bomb: true, Enclosures: 3, SpreadClicks: 8, Shields: 30, LastFill: fill}
}

type players struct {
	all     []bonuses.Holder
	failing map[bonuses.Holder]error
}

func (p players) Players(_ context.Context, page fills.Page) ([]bonuses.Holder, error) {
	if err := p.failing[page.After]; err != nil {
		return nil, err
	}
	var after []bonuses.Holder
	for _, holder := range p.all {
		if holder > page.After && len(after) < page.Size {
			after = append(after, holder)
		}
	}
	return after, nil
}

type notice struct {
	told int
}

func (n *notice) PublishFilled() {
	n.told++
}

type world struct {
	fills  *fills.MemoryStore
	hands  *bonuses.MemoryStore
	notice *notice
	clock  *cptime.FixedClock
}

func newWorld() world {
	return world{fills: fills.NewMemoryStore(), hands: bonuses.NewMemoryStore(), notice: &notice{}, clock: cptime.NewFixedClock(at)}
}

func (w world) run(t *testing.T, playing players) ([]fills.Fill, error) {
	t.Helper()

	return run_fills_usecase.New(w.fills, playing, bonuses.NewCharges(w.hands, sizes), w.notice, w.clock, 2).
		Execute(t.Context())
}

func snapshots(finished []fills.Fill) []fills.Snapshot {
	kept := make([]fills.Snapshot, 0, len(finished))
	for _, fill := range finished {
		kept = append(kept, fill.TakeSnapshot())
	}
	return kept
}

func TestEveryPlayerGetsAFullHandPageByPageAndTheFillIsKeptFinished(t *testing.T) {
	w := newWorld()
	w.fills.Keep(fills.Snapshot{ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at})
	w.hands.Keep(bob, bonuses.Held{Shields: 4})
	w.clock.Advance(time.Second)

	finished, err := w.run(t, players{all: []bonuses.Holder{ada, bob, cleo}})

	require.NoError(t, err)
	for _, holder := range []bonuses.Holder{ada, bob, cleo} {
		assert.Equal(t, full("season-0:finale:fill"), w.hands.Kept(holder), holder)
	}
	done := fills.Snapshot{
		ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at,
		Reached: cleo, Filled: 3, FinishedAt: at.Add(time.Second),
	}
	assert.Equal(t, []fills.Snapshot{done}, w.fills.Kept())
	assert.Equal(t, []fills.Snapshot{done}, snapshots(finished))
	assert.Equal(t, 2, w.notice.told, "the streams are told after each page")
}

func TestAFillCutShortGoesOnFromWhereItGotAndGivesNobodyTwice(t *testing.T) {
	w := newWorld()
	w.fills.Keep(fills.Snapshot{
		ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at, Reached: ada, Filled: 1,
	})
	spent, _ := full("season-0:finale:fill").AfterBomb()
	w.hands.Keep(ada, spent)
	w.hands.Keep(bob, spent)

	_, err := w.run(t, players{all: []bonuses.Holder{ada, bob, cleo}})

	require.NoError(t, err)
	assert.Equal(t, spent, w.hands.Kept(ada), "behind where the fill got")
	assert.Equal(t, spent, w.hands.Kept(bob), "filled before the restart, though the fill had not kept it got there")
	assert.Equal(t, full("season-0:finale:fill"), w.hands.Kept(cleo))
	assert.Equal(t, 3, w.fills.Kept()[0].Filled)
}

func TestFillsRunOneAtATimeOldestFirst(t *testing.T) {
	w := newWorld()
	w.fills.Keep(fills.Snapshot{ID: "later", Source: fills.SourceOperator, StartedAt: at.Add(time.Minute)})
	w.fills.Keep(fills.Snapshot{ID: "first", Source: fills.SourceInternal, StartedAt: at})

	finished, err := w.run(t, players{all: []bonuses.Holder{ada}})

	require.NoError(t, err)
	order := make([]bonuses.FillID, 0, len(finished))
	for _, kept := range snapshots(finished) {
		order = append(order, kept.ID)
	}
	assert.Equal(t, []bonuses.FillID{"first", "later"}, order)
	assert.Equal(t, full("later"), w.hands.Kept(ada))
}

func TestAFailureKeepsHowFarTheFillGot(t *testing.T) {
	w := newWorld()
	w.fills.Keep(fills.Snapshot{ID: "a1b2", Source: fills.SourceOperator, StartedAt: at})
	down := errors.New("postgres is down")

	finished, err := w.run(t, players{all: []bonuses.Holder{ada, bob, cleo}, failing: map[bonuses.Holder]error{bob: down}})

	require.ErrorIs(t, err, down)
	assert.Empty(t, finished)
	assert.Equal(t, []fills.Snapshot{{ID: "a1b2", Source: fills.SourceOperator, StartedAt: at, Reached: bob, Filled: 2}},
		w.fills.Kept())
	assert.True(t, w.hands.Kept(cleo).Empty())
}

func TestNothingToRunTellsNobody(t *testing.T) {
	w := newWorld()
	w.fills.Keep(fills.Snapshot{ID: "done", Source: fills.SourceOperator, StartedAt: at, FinishedAt: at})

	finished, err := w.run(t, players{all: []bonuses.Holder{ada}})

	require.NoError(t, err)
	assert.Empty(t, finished)
	assert.True(t, w.hands.Kept(ada).Empty())
	assert.Zero(t, w.notice.told)
	assert.Equal(t, []fills.Snapshot{{ID: "done", Source: fills.SourceOperator, StartedAt: at, FinishedAt: at}}, w.fills.Kept())
}
