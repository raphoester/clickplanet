package mutes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at       = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	bully    = messages.AccountID{15: 1}
	newGuest = messages.AccountID{15: 2}
)

func TestAnIPv6AddressIsMutedWithItsSlash64AndNeverWider(t *testing.T) {
	assert.Equal(t, mutes.Scope("2a01:e0a:1:2::/64"), mutes.ScopeOf("2a01:e0a:1:2:aaaa:bbbb:cccc:dddd"))
	assert.Equal(t, mutes.ScopeOf("2a01:e0a:1:2::1"), mutes.ScopeOf("2a01:e0a:1:2:ffff::9"))
	assert.NotEqual(t, mutes.ScopeOf("2a01:e0a:1:2::1"), mutes.ScopeOf("2a01:e0a:1:3::1"),
		"the next /64 of the same /32 is someone else")
}

func TestAnIPv4AddressIsMutedExactly(t *testing.T) {
	assert.Equal(t, mutes.Scope("203.0.113.7"), mutes.ScopeOf("203.0.113.7"))
	assert.Equal(t, mutes.Scope("203.0.113.7"), mutes.ScopeOf("::ffff:203.0.113.7"))
	assert.NotEqual(t, mutes.ScopeOf("203.0.113.7"), mutes.ScopeOf("203.0.113.8"))
}

func TestWhatIsNotAnAddressIsNoNetwork(t *testing.T) {
	for _, ip := range []string{"", "garbage", "203.0.113.7:443"} {
		assert.Equal(t, mutes.NoScope, mutes.ScopeOf(ip), ip)
	}
}

func TestNoDurationIsOneHour(t *testing.T) {
	duration, err := mutes.DurationOf(0)

	require.NoError(t, err)
	assert.Equal(t, time.Hour, duration)
}

func TestADurationIsKeptAsAsked(t *testing.T) {
	duration, err := mutes.DurationOf(90 * time.Minute)

	require.NoError(t, err)
	assert.Equal(t, 90*time.Minute, duration)
}

func TestANegativeOrFractionalDurationIsRefused(t *testing.T) {
	for _, asked := range []time.Duration{-time.Hour, 1500 * time.Millisecond, time.Millisecond} {
		_, err := mutes.DurationOf(asked)
		assert.ErrorIs(t, err, mutes.ErrInvalidDuration, asked.String())
	}
}

func TestAMuteEndsAfterItsDuration(t *testing.T) {
	mute := mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, "203.0.113.7"), at, time.Hour)

	assert.Equal(t, at.Add(time.Hour), mute.Until())
	assert.True(t, mute.ApplicableTo(mutes.NewCaller(bully, "198.51.100.1"), at.Add(59*time.Minute)))
	assert.False(t, mute.ApplicableTo(mutes.NewCaller(bully, "203.0.113.7"), at.Add(time.Hour)))
}

func TestAMuteHoldsAFreshGuestOnTheSameSlash64(t *testing.T) {
	mute := mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, "2a01:e0a:1:2::1"), at, time.Hour)

	assert.True(t, mute.ApplicableTo(mutes.NewCaller(newGuest, "2a01:e0a:1:2:dead:beef::5"), at))
	assert.False(t, mute.ApplicableTo(mutes.NewCaller(newGuest, "2a01:e0a:1:3::1"), at))
}

func TestAMuteWithNoNetworkHoldsNoOneElse(t *testing.T) {
	mute := mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, ""), at, time.Hour)

	assert.False(t, mute.ApplicableTo(mutes.NewCaller(newGuest, ""), at))
}

func TestARefusalSaysUntilWhenAndIsAMute(t *testing.T) {
	refusal := mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, ""), at, time.Hour).Refusal()

	require.ErrorIs(t, refusal, mutes.ErrMuted)
	assert.Equal(t, at.Add(time.Hour), refusal.Until())
	assert.Contains(t, refusal.Error(), "2026-10-07T21:00:00Z")
}

func TestTheBookLetsSpeakWhomNoMuteHolds(t *testing.T) {
	book := mutes.NewBook(inmemory_mute_storage.New(), cptime.NewFixedClock(at))

	assert.NoError(t, book.MuteError(t.Context(), mutes.NewCaller(bully, "203.0.113.7")))
}

func TestTheBookRefusesAMutedCallerWithTheMuteThatEndsLast(t *testing.T) {
	store := inmemory_mute_storage.New()
	require.NoError(t, store.Save(t.Context(), mutes.NewMute(mutes.MuteID{1}, mutes.NewCaller(bully, "2a01:e0a:1:2::1"), at, time.Hour)))
	require.NoError(t, store.Save(t.Context(), mutes.NewMute(mutes.MuteID{2}, mutes.NewCaller(bully, ""), at, 2*time.Hour)))
	book := mutes.NewBook(store, cptime.NewFixedClock(at.Add(time.Minute)))

	err := book.MuteError(t.Context(), mutes.NewCaller(bully, "2a01:e0a:1:2::1"))

	var refusal mutes.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, at.Add(2*time.Hour), refusal.Until())
}

type brokenReader struct{}

func (brokenReader) Mute(context.Context, mutes.Caller, time.Time) (mutes.Mute, error) {
	return mutes.Mute{}, errors.New("postgres is down")
}

func TestTheBookFailsWhenItCannotRead(t *testing.T) {
	err := mutes.NewBook(brokenReader{}, cptime.NewFixedClock(at)).MuteError(t.Context(), mutes.NewCaller(bully, ""))

	require.Error(t, err)
	assert.NotErrorIs(t, err, mutes.ErrMuted, "a read that failed says nothing about a mute")
}
