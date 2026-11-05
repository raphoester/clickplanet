package audit_release_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/audit_release"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubBook struct {
	err      error
	released []allegiances.Account
}

func (s *stubBook) Release(_ context.Context, account allegiances.Account) error {
	s.released = append(s.released, account)
	return s.err
}

func TestAReleaseIsLoggedWithTheAccount(t *testing.T) {
	var logs bytes.Buffer
	inner := &stubBook{}

	require.NoError(t, audit_release.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Release(t.Context(), ada))

	assert.Equal(t, []allegiances.Account{ada}, inner.released)
	assert.Contains(t, logs.String(), `level=WARN msg="admin allegiance release" account=`+ada)
}

func TestAFailedReleaseIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("postgres is down")

	err := audit_release.New(&stubBook{err: cause}, slog.New(slog.NewTextHandler(&logs, nil))).Release(t.Context(), ada)

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `msg="admin allegiance release failed" account=`+ada)
}
