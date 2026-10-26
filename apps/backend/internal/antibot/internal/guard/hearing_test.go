package guard_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type lateFeed struct {
	handled *atomic.Bool
}

func (f lateFeed) Name() string { return "events:late" }

func (f lateFeed) Run(ctx context.Context) {
	<-ctx.Done()
	time.Sleep(20 * time.Millisecond)
	f.handled.Store(true)
}

type witness struct {
	handled *atomic.Bool
	saved   []bool
}

func (w *witness) Name() string { return "witness" }

func (w *witness) Save() ([]byte, error) {
	w.saved = append(w.saved, w.handled.Load())
	return evidence.Encode(w.handled.Load())
}

func (w *witness) Load([]byte) error { return nil }

func (w *witness) Forget(time.Time) {}

func TestTheLastFlushWaitsForWhatTheFeedsStillHandle(t *testing.T) {
	handled := &atomic.Bool{}
	seen := &witness{handled: handled}
	config := guard.Config{}
	config.Sequencer.Enabled = true
	set, err := rules.Seeded().Restore()
	require.NoError(t, err)
	inForce := rules.NewInForce()
	inForce.Enforce(set)
	watchdogs, err := guard.NewInMemory(config, inForce, cptime.NewFixedClock(time.Now()), guard.Observer{},
		bans.NewMemoryStore(), bans.NewMemoryStore(), &bans.MemoryEnforcer{}, evidence.NewMemoryPersistence(), seen)
	require.NoError(t, err)

	hearing := watchdogs.Hearing(lateFeed{handled: handled})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	hearing.Run(ctx)

	require.NotEmpty(t, seen.saved)
	assert.True(t, seen.saved[len(seen.saved)-1], "the evidence saved last holds what the feed handled on its way out")
	assert.Equal(t, "antibot+events:late", hearing.Name())
}
