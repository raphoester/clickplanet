package ban_player_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/ban_player"
)

var until = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

type banner struct {
	scopes    []string
	durations []time.Duration
}

func (b *banner) Ban(scope string, duration time.Duration) antibot.Sentence {
	b.scopes = append(b.scopes, scope)
	b.durations = append(b.durations, duration)
	return antibot.Sentence{Offence: 1, Until: until}
}

func (b *banner) Enforcing() bool { return true }

func TestAnAddressIsBannedAsItsScope(t *testing.T) {
	b := &banner{}

	out, err := ban_player.New(b).Execute(t.Context(), ban_player.In{Scope: "2001:db8:1:2::9", Duration: time.Hour})
	require.NoError(t, err)

	assert.Equal(t, ban_player.Out{Scope: "2001:db8:1:2::/64", Offence: 1, Until: until, Enforced: true}, out)
	assert.Equal(t, []string{"2001:db8:1:2::/64"}, b.scopes)
	assert.Equal(t, []time.Duration{time.Hour}, b.durations)
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	b := &banner{}

	_, err := ban_player.New(b).Execute(t.Context(), ban_player.In{Scope: "1.2.3.0/24"})
	require.ErrorIs(t, err, clicks.ErrInvalidScope)

	_, err = ban_player.New(b).Execute(t.Context(), ban_player.In{Scope: "1.2.3.4", Duration: -time.Hour})
	require.ErrorIs(t, err, ban_player.ErrNegativeDuration)

	assert.Empty(t, b.scopes)
}

func TestWithTheAntiBotOffThereIsNothingToBanWith(t *testing.T) {
	_, err := ban_player.New(nil).Execute(t.Context(), ban_player.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, ban_player.ErrAntiBotOff)
}
