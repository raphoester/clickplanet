package unban_player_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/unban_player_usecase"
)

var until = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

type unbanner struct {
	banned   map[string]antibot.Sentence
	scopes   []string
	accounts []string
	off      bool
	failing  error
}

func bannedOn(keys ...string) *unbanner {
	u := &unbanner{banned: map[string]antibot.Sentence{}}
	for _, key := range keys {
		u.banned[key] = antibot.Sentence{Flags: 2, Offence: 1, Until: until}
	}
	return u
}

func (u *unbanner) Sentence(_ context.Context, scope, account string) (antibot.Sentence, bool, error) {
	sentence, ok := u.banned[scope+account]
	return sentence, ok, u.failing
}

func (u *unbanner) Unban(_ context.Context, scope, account string) error {
	u.scopes = append(u.scopes, scope)
	u.accounts = append(u.accounts, account)
	return nil
}

func (u *unbanner) Enabled() bool { return !u.off }

func TestAnAddressIsUnbannedAsItsScope(t *testing.T) {
	u := bannedOn("2001:db8:1:2::/64")

	out, err := unban_player_usecase.New(u).Execute(t.Context(), unban_player_usecase.In{Scope: "2001:db8:1:2::9"})
	require.NoError(t, err)

	assert.Equal(t, unban_player_usecase.Out{Scope: "2001:db8:1:2::/64", Offence: 1, Until: until}, out)
	assert.Equal(t, []string{"2001:db8:1:2::/64"}, u.scopes)
	assert.Equal(t, []string{""}, u.accounts)
}

func TestAnAccountIsUnbannedAlone(t *testing.T) {
	u := bannedOn("0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")

	out, err := unban_player_usecase.New(u).Execute(t.Context(),
		unban_player_usecase.In{Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10"})
	require.NoError(t, err)

	assert.Equal(t, "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", out.Account)
	assert.Empty(t, out.Scope)
	assert.Equal(t, []string{""}, u.scopes)
	assert.Equal(t, []string{"0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}, u.accounts)
}

func TestACallerWithNoRunningBanIsRefusedAndNothingIsLifted(t *testing.T) {
	u := bannedOn()

	_, err := unban_player_usecase.New(u).Execute(t.Context(), unban_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, unban_player_usecase.ErrNotBanned)

	assert.Empty(t, u.scopes)
}

func TestItRefusesBothOrNeitherAndWhatIsNeitherAScopeNorAnAccount(t *testing.T) {
	u := bannedOn("1.2.3.4")

	for in, want := range map[unban_player_usecase.In]error{
		{}: ledger.ErrNoCaller,
		{Scope: "1.2.3.4", Account: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}: ledger.ErrNoCaller,
		{Scope: "1.2.3.0/24"}: ledger.ErrInvalidScope,
		{Account: "guest"}:    ledger.ErrInvalidAccount,
	} {
		_, err := unban_player_usecase.New(u).Execute(t.Context(), in)
		require.ErrorIs(t, err, want)
	}

	assert.Empty(t, u.scopes)
}

func TestWithTheAntiBotOffThereIsNoBanToLift(t *testing.T) {
	u := bannedOn("1.2.3.4")
	u.off = true

	_, err := unban_player_usecase.New(u).Execute(t.Context(), unban_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, unban_player_usecase.ErrAntiBotOff)

	assert.Empty(t, u.scopes)
}

func TestABanThatCannotBeReadIsAnErrorAndNothingIsLifted(t *testing.T) {
	u := bannedOn("1.2.3.4")
	cause := errors.New("postgres is down")
	u.failing = cause

	_, err := unban_player_usecase.New(u).Execute(t.Context(), unban_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, cause)

	assert.Empty(t, u.scopes)
}
