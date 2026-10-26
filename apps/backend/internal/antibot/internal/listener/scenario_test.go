package listener_test

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/listener"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type watched struct {
	clock    *cptime.FixedClock
	tries    func(*planetv1.ClickTried) error
	enforcer *bans.MemoryEnforcer
}

func watch(t *testing.T) watched {
	t.Helper()

	config := guard.Config{}
	config.Metronome.TrackWindow = 15 * time.Minute

	kept := rules.Seeded()
	only := kept.Rules
	only.Bans.BanDurations = []*durationpb.Duration{durationpb.New(24 * time.Hour)}
	only.Retaker.Enabled, only.Sequencer.Enabled, only.Defender.Enabled = false, false, false
	only.Catcher.Enabled, only.Cohort.Enabled, only.Scraper.Enabled = false, false, false
	only.Churner.Enabled, only.Hopper.Enabled, only.Rotator.Enabled = false, false, false
	set, err := kept.Restore(guard.Parts())
	require.NoError(t, err)
	inForce := rules.NewInForce()
	inForce.Enforce(set)

	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 2, 0, 0, 0, time.UTC))
	enforcer := &bans.MemoryEnforcer{}
	marks := listener.NewMarks()
	watchdogs := guard.NewInMemory(config, inForce, clock, guard.Observer{}, bans.NewMemoryStore(), bans.NewMemoryStore(),
		enforcer, evidence.NewMemoryPersistence(), marks)
	require.NoError(t, watchdogs.Load(t.Context()))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		watchdogs.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	heard := listener.New(watchdogs, marks).ClickTried()
	return watched{
		clock:    clock,
		tries:    func(fact *planetv1.ClickTried) error { return heard.Handle(t.Context(), fact) },
		enforcer: enforcer,
	}
}

func (w watched) try(outcome planetv1.ClickOutcome, tile uint32) *planetv1.ClickTried {
	return &planetv1.ClickTried{
		Scope: "2001:db8:e487::/64", TileId: tile, Country: "bg", Outcome: outcome, TriedAt: timestamppb.New(w.clock.Now()),
	}
}

func loopIntoTheThrottle(t *testing.T, w watched, toldThrottled bool) bool {
	t.Helper()

	//nolint:gosec // seeded test PRNG
	random := rand.New(rand.NewPCG(9, 10))

	for range 4000 {
		w.clock.Advance(time.Duration(890+random.IntN(121)) * time.Millisecond)

		outcome := planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN
		if random.IntN(2) == 0 {
			if !toldThrottled {
				continue
			}
			outcome = planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED
		}
		require.NoError(t, w.tries(w.try(outcome, 180000+uint32(random.IntN(60000)))))

		if len(w.enforcer.Told()) > 0 {
			return true
		}
	}
	return false
}

func TestTheLoopOfSeptember14IsRefusedThroughItsToldTries(t *testing.T) {
	w := watch(t)

	require.True(t, loopIntoTheThrottle(t, w, true), "every try, the 429s included, keeps the beat at production's bounds")

	told := w.enforcer.Told()
	require.Len(t, told, 1)
	assert.Equal(t, bans.Caller{Scope: "2001:db8:e487::/64"}, told[0].Caller)
	assert.Equal(t, told[0].At.Add(24*time.Hour), told[0].Until, "planet refuses the loop for the first step of the ladder")
}

func TestTheSameLoopToldOnlyByItsTakesIsMissed(t *testing.T) {
	assert.False(t, loopIntoTheThrottle(t, watch(t), false),
		"the takes alone are 0.95s, 1.9s and 2.85s apart: a spread no clock shows")
}

func TestARefusedCallersTriesFeedNoWatchdog(t *testing.T) {
	w := watch(t)

	for range 2000 {
		w.clock.Advance(time.Second)
		require.NoError(t, w.tries(w.try(planetv1.ClickOutcome_CLICK_OUTCOME_REFUSED, 1)))
	}

	assert.Empty(t, w.enforcer.Told(), "a ban does not feed itself")
}
