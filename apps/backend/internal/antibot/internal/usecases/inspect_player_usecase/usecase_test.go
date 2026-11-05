package inspect_player_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/inspect_player_usecase"
)

type examiner struct {
	asked []string
}

func (e *examiner) Examine(_ context.Context, scope, account string) (guard.Examination, error) {
	e.asked = append(e.asked, scope+account)
	return guard.Examination{Scope: scope, Account: account, Clicks: 3}, nil
}

func TestAnAddressIsInspectedAsItsScope(t *testing.T) {
	e := &examiner{}

	out, err := inspect_player_usecase.New(e).Execute(t.Context(), inspect_player_usecase.In{Scope: "2001:db8:1:2::9"})
	require.NoError(t, err)

	assert.Equal(t, guard.Examination{Scope: "2001:db8:1:2::/64", Clicks: 3}, out)
	assert.Equal(t, []string{"2001:db8:1:2::/64"}, e.asked)
}

func TestAnAccountIsInspectedByItsID(t *testing.T) {
	e := &examiner{}

	_, err := inspect_player_usecase.New(e).Execute(t.Context(),
		inspect_player_usecase.In{Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10"})
	require.NoError(t, err)

	assert.Equal(t, []string{"0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}, e.asked, "the guard finds the scope it clicked from last")
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	e := &examiner{}

	_, err := inspect_player_usecase.New(e).Execute(t.Context(), inspect_player_usecase.In{Scope: "1.2.3.0/24"})
	require.ErrorIs(t, err, bans.ErrInvalidScope)

	assert.Empty(t, e.asked)
}
