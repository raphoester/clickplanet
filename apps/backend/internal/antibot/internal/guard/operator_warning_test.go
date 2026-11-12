package guard_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
)

func TestAnOperatorsWarningBansTheRingOnTwoHoursOfTurnsAfterIt(t *testing.T) {
	s := newStack(t)
	s.signIn("lynx", "zebra")

	//nolint:gosec // seeded test PRNG
	r := rand.New(rand.NewPCG(11, 4))
	tabs := [2]func() (string, string){
		func() (string, string) { return russianAddress, "lynx" },
		func() (string, string) { return russianAddress, "zebra" },
	}

	s.switchTabs(r, 2*time.Hour, tabs, 4*time.Second, 10*time.Second)
	require.Empty(t, s.warnings, "under four hours of turns")
	require.Empty(t, s.reports, "a ring never warned is not banned on a suspect reading")

	warning, given, err := s.guard.Warn(t.Context(), "lynx")
	require.NoError(t, err)
	require.True(t, given)
	assert.Equal(t, []warnings.Warning{warning}, s.feed.Published(), "the page shows it at once")

	s.switchTabs(r, 110*time.Minute, tabs, 4*time.Second, 10*time.Second)
	require.Empty(t, s.reports, "the turns before the warning never ban")

	s.switchTabs(r, 30*time.Minute, tabs, 4*time.Second, 10*time.Second)

	assert.ElementsMatch(t, []string{"lynx", "zebra"}, s.reported(), "the ring falls with the account warned")
	assert.WithinRange(t, s.clock.Now(), warning.At.Add(115*time.Minute), warning.At.Add(130*time.Minute),
		"24 slots of turns after the slot of the warning")
	assert.Empty(t, s.warnings, "the jury warns no ring that holds a warned account")
	assert.Equal(t, warning.At, s.examine("", "lynx").WarnedAt)
}

func TestAnOperatorWarnsAnAccountOnce(t *testing.T) {
	s := newStack(t)

	first, given, err := s.guard.Warn(t.Context(), "lynx")
	require.NoError(t, err)
	require.True(t, given)

	s.clock.Advance(time.Hour)
	_, given, err = s.guard.Warn(t.Context(), "lynx")
	require.NoError(t, err)

	assert.False(t, given)
	kept, warned, err := s.guard.Warning(t.Context(), "lynx")
	require.NoError(t, err)
	require.True(t, warned)
	assert.Equal(t, first.At, kept.At, "the first warning is kept")
	assert.Len(t, s.feed.Published(), 1, "shown once")
}

func TestAnOperatorWarnsNobodyWhileTheBansAreNotEnforced(t *testing.T) {
	s := newStack(t)
	s.set.Bans.Enforce = proto.Bool(false)
	s.play(s.set)

	_, given, err := s.guard.Warn(t.Context(), "lynx")
	require.NoError(t, err)

	assert.False(t, given)
	_, warned, err := s.guard.Warning(t.Context(), "lynx")
	require.NoError(t, err)
	assert.False(t, warned, "kept, it would ban a ring that never saw it")
	assert.Empty(t, s.feed.Published())
}
