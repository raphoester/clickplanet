package inspect_player_usecase_test

import (
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

func (e *examiner) Examine(scope string) antibot.Examination {
	e.scopes = append(e.scopes, scope)
	return antibot.Examination{Scope: scope, Clicks: 3}
}

func (e *examiner) Enabled() bool { return !e.off }

func TestAnAddressIsInspectedAsItsScope(t *testing.T) {
	e := &examiner{}

	out, err := inspect_player_usecase.New(e).Execute(t.Context(), inspect_player_usecase.In{Scope: "2001:db8:1:2::9"})
	require.NoError(t, err)

	assert.Equal(t, antibot.Examination{Scope: "2001:db8:1:2::/64", Clicks: 3}, out)
	assert.Equal(t, []string{"2001:db8:1:2::/64"}, e.scopes)
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	e := &examiner{}

	_, err := inspect_player_usecase.New(e).Execute(t.Context(), inspect_player_usecase.In{Scope: "1.2.3.0/24"})
	require.ErrorIs(t, err, ledger.ErrInvalidScope)

	assert.Empty(t, e.scopes)
}

func TestWithTheAntiBotOffThereIsNothingToInspect(t *testing.T) {
	_, err := inspect_player_usecase.New(&examiner{off: true}).Execute(t.Context(), inspect_player_usecase.In{Scope: "1.2.3.4"})
	require.ErrorIs(t, err, inspect_player_usecase.ErrAntiBotOff)
}
