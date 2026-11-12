package warn_player_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/warn_player_usecase"
)

const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

var now = time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)

type warner struct {
	enforcing bool
	// Stops enforcing between the check and the warning, as a rule set played then would.
	stopsAtWarn bool
	failing     error
	kept        map[string]guard.Warning
	asked       []string
}

func newWarner() *warner {
	return &warner{enforcing: true, kept: map[string]guard.Warning{}}
}

func (w *warner) Enforcing() bool { return w.enforcing }

func (w *warner) Warn(_ context.Context, account string) (guard.Warning, bool, error) {
	w.asked = append(w.asked, account)
	if w.stopsAtWarn {
		w.enforcing = false
	}
	if w.failing != nil || !w.enforcing {
		return guard.Warning{}, false, w.failing
	}
	if _, warned := w.kept[account]; warned {
		return guard.Warning{}, false, nil
	}
	w.kept[account] = guard.Warning{Account: account, At: now}
	return w.kept[account], true, nil
}

func (w *warner) Warning(_ context.Context, account string) (guard.Warning, bool, error) {
	warning, warned := w.kept[account]
	return warning, warned, nil
}

func TestAnAccountIsWarned(t *testing.T) {
	w := newWarner()

	out, err := warn_player_usecase.New(w).Execute(t.Context(), warn_player_usecase.In{Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10"})
	require.NoError(t, err)

	assert.Equal(t, warn_player_usecase.Out{Account: account, WarnedAt: now}, out)
	assert.Equal(t, []string{account}, w.asked)
}

func TestAnAccountWarnedBeforeAnswersItsFirstWarning(t *testing.T) {
	w := newWarner()
	first := now.Add(-48 * time.Hour)
	acknowledged := first.Add(time.Minute)
	w.kept[account] = guard.Warning{Account: account, At: first, AcknowledgedAt: acknowledged}

	out, err := warn_player_usecase.New(w).Execute(t.Context(), warn_player_usecase.In{Account: account})
	require.NoError(t, err)

	assert.Equal(t, warn_player_usecase.Out{
		Account: account, WarnedAt: first, AlreadyWarned: true, AcknowledgedAt: acknowledged,
	}, out)
}

func TestNobodyIsWarnedWhileTheBansAreNotEnforced(t *testing.T) {
	w := newWarner()
	w.enforcing = false

	_, err := warn_player_usecase.New(w).Execute(t.Context(), warn_player_usecase.In{Account: account})

	require.ErrorIs(t, err, warn_player_usecase.ErrNotEnforced)
	assert.Empty(t, w.asked)
}

func TestARuleSetThatStopsEnforcingMidCallIsNotEnforced(t *testing.T) {
	w := newWarner()
	w.stopsAtWarn = true

	_, err := warn_player_usecase.New(w).Execute(t.Context(), warn_player_usecase.In{Account: account})

	require.ErrorIs(t, err, warn_player_usecase.ErrNotEnforced)
	assert.Empty(t, w.kept)
}

func TestItRefusesWhatIsNotAnAccount(t *testing.T) {
	w := newWarner()

	for _, typed := range []string{"", "not-an-account", "192.0.2.1"} {
		_, err := warn_player_usecase.New(w).Execute(t.Context(), warn_player_usecase.In{Account: typed})
		require.ErrorIs(t, err, bans.ErrInvalidAccount, typed)
	}

	assert.Empty(t, w.asked)
}

func TestAWarningThatCannotBeKeptFails(t *testing.T) {
	w := newWarner()
	cause := errors.New("postgres down")
	w.failing = cause

	_, err := warn_player_usecase.New(w).Execute(t.Context(), warn_player_usecase.In{Account: account})

	require.ErrorIs(t, err, cause)
	assert.NotErrorIs(t, err, warn_player_usecase.ErrNotEnforced)
}
