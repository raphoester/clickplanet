package unban_player_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/unban_player_usecase"
)

var (
	until         = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	pardonedUntil = until.Add(72 * time.Hour)
)

type unbanner struct {
	banned   map[string]guard.Sentence
	pardoned map[string]guard.Pardon
	scopes   []string
	accounts []string
	failing  error
}

func bannedOn(keys ...string) *unbanner {
	u := &unbanner{banned: map[string]guard.Sentence{}, pardoned: map[string]guard.Pardon{}}
	for _, key := range keys {
		u.banned[key] = guard.Sentence{Flags: 2, Offence: 1, Until: until}
	}
	return u
}

func (u *unbanner) LastSentence(_ context.Context, scope, account string) (guard.Sentence, bool, error) {
	sentence, ok := u.banned[scope+account]
	return sentence, ok, u.failing
}

func (u *unbanner) Unban(_ context.Context, scope, account string) error {
	u.scopes = append(u.scopes, scope)
	u.accounts = append(u.accounts, account)
	u.pardoned[scope+account] = guard.Pardon{Lifted: u.banned[scope+account], Until: pardonedUntil}
	return nil
}

func (u *unbanner) Pardon(_ context.Context, scope, account string) (guard.Pardon, bool, error) {
	pardon, ok := u.pardoned[scope+account]
	return pardon, ok, nil
}

func TestAnAddressIsUnbannedAsItsScope(t *testing.T) {
	u := bannedOn("2001:db8:1:2::/64")

	out, err := unban_player_usecase.New(u).Execute(t.Context(), unban_player_usecase.In{Scope: "2001:db8:1:2::9"})
	require.NoError(t, err)

	assert.Equal(t, unban_player_usecase.Out{Scope: "2001:db8:1:2::/64", Offence: 1, Until: until, PardonedUntil: pardonedUntil}, out)
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
	assert.Equal(t, pardonedUntil, out.PardonedUntil)
	assert.Equal(t, []string{""}, u.scopes)
	assert.Equal(t, []string{"0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}, u.accounts)
}

func TestACallerWithNoBanToLiftIsRefusedAndNothingIsLifted(t *testing.T) {
	u := bannedOn()

	_, err := unban_player_usecase.New(u).Execute(t.Context(), unban_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, unban_player_usecase.ErrNotBanned)

	assert.Empty(t, u.scopes)
}

func TestItRefusesBothOrNeitherAndWhatIsNeitherAScopeNorAnAccount(t *testing.T) {
	u := bannedOn("1.2.3.4")

	for in, want := range map[unban_player_usecase.In]error{
		{}: bans.ErrNoCaller,
		{Scope: "1.2.3.4", Account: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}: bans.ErrNoCaller,
		{Scope: "1.2.3.0/24"}: bans.ErrInvalidScope,
		{Account: "guest"}:    bans.ErrInvalidAccount,
	} {
		_, err := unban_player_usecase.New(u).Execute(t.Context(), in)
		require.ErrorIs(t, err, want)
	}

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
