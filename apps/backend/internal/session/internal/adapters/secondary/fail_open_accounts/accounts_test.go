package fail_open_accounts_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/secondary/fail_open_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

type fixedAccounts struct {
	resolution domain.Resolution
	err        error
}

func (a fixedAccounts) Resolve(context.Context, string, bool) (domain.Resolution, error) {
	return a.resolution, a.err
}

func resolve(t *testing.T, next domain.Accounts) (domain.Resolution, *prometheus.Registry) {
	t.Helper()

	registry := prometheus.NewRegistry()
	accounts := fail_open_accounts.New(next, slog.New(slog.DiscardHandler), registry)

	resolution, err := accounts.Resolve(t.Context(), "cp_sid=abc", true)
	require.NoError(t, err)

	return resolution, registry
}

func assertCounted(t *testing.T, registry *prometheus.Registry, outcome string) {
	t.Helper()

	expected := `
# HELP session_account_resolutions_total Account resolutions at mint, by outcome: account, none, or failed (minted with no account)
# TYPE session_account_resolutions_total counter
session_account_resolutions_total{outcome="` + outcome + `"} 1
`
	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(expected)))
}

func TestAFailureMintsWithNoAccountAndIsCounted(t *testing.T) {
	resolution, registry := resolve(t, fixedAccounts{err: errors.New("auth is down")})

	assert.Equal(t, domain.Resolution{}, resolution)
	assertCounted(t, registry, "failed")
}

func TestAnAccountIsPassedOnAndCounted(t *testing.T) {
	account := domain.Resolution{Account: uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"), SetCookie: "cp_sid=x"}

	resolution, registry := resolve(t, fixedAccounts{resolution: account})

	assert.Equal(t, account, resolution)
	assertCounted(t, registry, "account")
}

func TestNoAccountIsCountedAsNone(t *testing.T) {
	_, registry := resolve(t, fixedAccounts{})

	assertCounted(t, registry, "none")
}
