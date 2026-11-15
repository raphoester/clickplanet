package guard_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sharedConnection = "203.0.113.54"

// Each click comes 20 seconds after the one before, on a tile of its own.
func (s *stack) playAs(scope, account string, clicks int) {
	for range clicks {
		s.clock.Advance(20 * time.Second)
		s.clickAs(scope, account, uint32(500000+s.clock.Now().Unix()%100000), "fr")
	}
}

func TestTwoAccountsPlayingInTurnsFromOneConnectionAreEachNoticedOnceAndNothingElse(t *testing.T) {
	s := newStack(t)

	s.playAs(sharedConnection, "main", 5)
	s.playAs(sharedConnection, "alt", 5)
	require.Empty(t, s.notices, "the alt only followed the main account so far")

	s.playAs(sharedConnection, "main", 5)
	s.playAs(sharedConnection, "alt", 5)
	s.playAs(sharedConnection, "main", 5)

	require.Len(t, s.notices, 2)
	assert.Equal(t, "main", s.notices[0].Account)
	assert.Equal(t, "alt", s.notices[0].Alongside)
	assert.Equal(t, "alt", s.notices[1].Account)
	assert.Equal(t, "main", s.notices[1].Alongside)
	assert.Equal(t, s.notices, s.noticeFeed.Published(), "the page shows each at once")
	assert.Empty(t, s.reports, "a notice bans nobody")
	assert.Empty(t, s.warnings, "nor warns anybody")
	assert.Empty(t, s.errors)

	examined := s.examine("", "main")
	assert.Equal(t, s.notices[0].At, examined.NoticedAt)
	assert.Equal(t, "alt", examined.NoticedAlongside)
	assert.True(t, examined.NoticeAcknowledgedAt.IsZero())
	assert.True(t, examined.WarnedAt.IsZero())
}

func TestAGuestThatSignsInToItsAccountIsNotNoticed(t *testing.T) {
	s := newStack(t)

	s.playAs(sharedConnection, "guest", 30)
	s.playAs(sharedConnection, "account", 30)

	assert.Empty(t, s.notices)
}

func TestAnAccountWarnedBeforeIsNotNoticed(t *testing.T) {
	s := newStack(t)
	_, given, err := s.guard.Warn(t.Context(), "main")
	require.NoError(t, err)
	require.True(t, given)

	s.playAs(sharedConnection, "main", 3)
	s.playAs(sharedConnection, "alt", 3)
	s.playAs(sharedConnection, "main", 3)
	s.playAs(sharedConnection, "alt", 3)

	require.Len(t, s.notices, 1, "it was told more than a notice says")
	assert.Equal(t, "alt", s.notices[0].Account)
}

func TestANoticeThatCannotBeKeptIsAnError(t *testing.T) {
	s := newStack(t)
	s.noticed.FailWith(errors.New("postgres is down"))

	s.playAs(sharedConnection, "main", 3)
	s.playAs(sharedConnection, "alt", 3)
	s.playAs(sharedConnection, "main", 3)

	assert.Empty(t, s.notices)
	assert.Empty(t, s.noticeFeed.Published())
	assert.NotEmpty(t, s.errors)
}
