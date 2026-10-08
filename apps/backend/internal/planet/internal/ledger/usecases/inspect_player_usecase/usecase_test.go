package inspect_player_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/inspect_player_usecase"
)

type examiner struct {
	scopes []string
	off    bool
}

func (e *examiner) Examine(_ context.Context, scope, account string) (antibot.Examination, error) {
	e.scopes = append(e.scopes, scope)
	return antibot.Examination{Scope: scope, Account: account, Clicks: 3}, nil
}

type book []ledger.Event

func (b book) Replay(see func(ledger.Event)) ledger.Position {
	for _, event := range b {
		see(event)
	}
	return ledger.Position(len(b))
}

func (e *examiner) Enabled() bool { return !e.off }

func TestAnAddressIsInspectedAsItsScope(t *testing.T) {
	e := &examiner{}

	out, err := inspect_player_usecase.New(e, book{}).Execute(t.Context(), inspect_player_usecase.In{Scope: "2001:db8:1:2::9"})
	require.NoError(t, err)

	assert.Equal(t, antibot.Examination{Scope: "2001:db8:1:2::/64", Clicks: 3}, out)
	assert.Equal(t, []string{"2001:db8:1:2::/64"}, e.scopes)
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	e := &examiner{}

	_, err := inspect_player_usecase.New(e, book{}).Execute(t.Context(), inspect_player_usecase.In{Scope: "1.2.3.0/24"})
	require.ErrorIs(t, err, ledger.ErrInvalidScope)

	assert.Empty(t, e.scopes)
}

func TestWithTheAntiBotOffThereIsNothingToInspect(t *testing.T) {
	_, err := inspect_player_usecase.New(&examiner{off: true}, book{}).Execute(t.Context(), inspect_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, inspect_player_usecase.ErrAntiBotOff)
}

func TestAnAccountIsInspectedOnTheScopeOfItsLatestTake(t *testing.T) {
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	e := &examiner{}
	takes := book{
		ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: guest},
		ledger.Taking{Tile: 2, Scope: "5.6.7.8", Account: guest},
		ledger.Taking{Tile: 3, Scope: "9.9.9.9", Account: "someone-else"},
	}

	out, err := inspect_player_usecase.New(e, takes).Execute(t.Context(), inspect_player_usecase.In{Account: guest})
	require.NoError(t, err)

	assert.Equal(t, antibot.Examination{Scope: "5.6.7.8", Account: guest, Clicks: 3}, out)
}

func TestAnAccountWithNoTakeIsInspectedOnItsBansAlone(t *testing.T) {
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	e := &examiner{}

	out, err := inspect_player_usecase.New(e, book{}).Execute(t.Context(), inspect_player_usecase.In{Account: guest})
	require.NoError(t, err)

	assert.Equal(t, guest, out.Account)
	assert.Equal(t, []string{""}, e.scopes)
}
