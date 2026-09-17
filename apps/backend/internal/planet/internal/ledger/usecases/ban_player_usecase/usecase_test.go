package ban_player_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/ban_player_usecase"
)

var until = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

type banner struct {
	scopes    []string
	accounts  []string
	durations []time.Duration
	off       bool
}

func (b *banner) Ban(scope, account string, duration time.Duration) antibot.Sentence {
	b.scopes = append(b.scopes, scope)
	b.accounts = append(b.accounts, account)
	b.durations = append(b.durations, duration)
	return antibot.Sentence{Offence: 1, Until: until}
}

func (b *banner) Enforcing() bool { return true }

func (b *banner) Enabled() bool { return !b.off }

func TestAnAddressIsBannedAsItsScope(t *testing.T) {
	b := &banner{}

	out, err := ban_player_usecase.New(b).Execute(t.Context(), ban_player_usecase.In{Scope: "2001:db8:1:2::9", Duration: time.Hour})
	require.NoError(t, err)

	assert.Equal(t, ban_player_usecase.Out{Scope: "2001:db8:1:2::/64", Offence: 1, Until: until, Enforced: true}, out)
	assert.Equal(t, []string{"2001:db8:1:2::/64"}, b.scopes)
	assert.Equal(t, []time.Duration{time.Hour}, b.durations)
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	b := &banner{}

	_, err := ban_player_usecase.New(b).Execute(t.Context(), ban_player_usecase.In{Scope: "1.2.3.0/24"})
	require.ErrorIs(t, err, ledger.ErrInvalidScope)

	_, err = ban_player_usecase.New(b).Execute(t.Context(), ban_player_usecase.In{Scope: "1.2.3.4", Duration: -time.Hour})
	require.ErrorIs(t, err, ban_player_usecase.ErrNegativeDuration)

	assert.Empty(t, b.scopes)
}

func TestWithTheAntiBotOffThereIsNothingToBanWith(t *testing.T) {
	_, err := ban_player_usecase.New(&banner{off: true}).Execute(t.Context(), ban_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, ban_player_usecase.ErrAntiBotOff)
}

func TestAnAccountIsBannedAlone(t *testing.T) {
	b := &banner{}

	out, err := ban_player_usecase.New(b).Execute(t.Context(),
		ban_player_usecase.In{Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10"})
	require.NoError(t, err)

	assert.Equal(t, "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", out.Account)
	assert.Empty(t, out.Scope)
	assert.Equal(t, []string{""}, b.scopes)
	assert.Equal(t, []string{"0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}, b.accounts)
}

func TestItRefusesBothOrNeitherAndWhatIsNotAnAccount(t *testing.T) {
	b := &banner{}

	_, err := ban_player_usecase.New(b).Execute(t.Context(), ban_player_usecase.In{})
	require.ErrorIs(t, err, ledger.ErrNoCaller)

	_, err = ban_player_usecase.New(b).Execute(t.Context(),
		ban_player_usecase.In{Scope: "1.2.3.4", Account: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"})
	require.ErrorIs(t, err, ledger.ErrNoCaller)

	_, err = ban_player_usecase.New(b).Execute(t.Context(), ban_player_usecase.In{Account: "guest"})
	require.ErrorIs(t, err, ledger.ErrInvalidAccount)

	assert.Empty(t, b.accounts)
}
