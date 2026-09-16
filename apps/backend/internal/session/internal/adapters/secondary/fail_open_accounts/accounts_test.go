package fail_open_accounts_test

import (
	"context"
	"errors"
	"fmt"
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
	resolution *domain.Resolution
	err        error
}

func (a fixedAccounts) Resolve(context.Context, string, bool) (*domain.Resolution, error) {
	return a.resolution, a.err
}

func resolve(registry *prometheus.Registry, next domain.Accounts) (*domain.Resolution, error) {
	resolution, err := fail_open_accounts.New(next, slog.New(slog.DiscardHandler), registry).
		Resolve(context.Background(), "cp_sid=abc", true)
	if err != nil {
		return nil, fmt.Errorf("resolve failed: %w", err)
	}
	return resolution, nil
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

func TestAFailureIsNoAccountAndIsCounted(t *testing.T) {
	registry := prometheus.NewRegistry()
	resolution, err := resolve(registry, fixedAccounts{err: errors.New("auth is down")})

	require.ErrorIs(t, err, domain.ErrNoAccount)
	assert.Nil(t, resolution)
	assertCounted(t, registry, "failed")
}

func TestAnAccountIsPassedOnAndCounted(t *testing.T) {
	account := &domain.Resolution{Account: uuid.UUID{15: 1}, SetCookie: "cp_sid=x"}

	registry := prometheus.NewRegistry()
	resolution, err := resolve(registry, fixedAccounts{resolution: account})

	require.NoError(t, err)
	assert.Equal(t, account, resolution)
	assertCounted(t, registry, "account")
}

func TestNoAccountIsPassedOnAndCountedAsNone(t *testing.T) {
	registry := prometheus.NewRegistry()
	resolution, err := resolve(registry, fixedAccounts{err: domain.ErrNoAccount})

	require.ErrorIs(t, err, domain.ErrNoAccount)
	assert.Nil(t, resolution)
	assertCounted(t, registry, "none")
}
