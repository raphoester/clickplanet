package refusals_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
)

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

func TestAnAdmissionEndsTheRefusalNow(t *testing.T) {
	refusal := refusals.Refusal{}.
		Decided(refusals.Refused(at.Add(time.Hour), at)).
		Decided(refusals.Admitted(at.Add(time.Minute)))

	assert.True(t, refusal.Standing(at.Add(time.Second)), "an admission does not reach back")
	assert.False(t, refusal.Standing(at.Add(time.Minute)))
}

func TestACallerIsAScopeOrAnAccountAndOnlyOne(t *testing.T) {
	scope, err := refusals.ParseCaller("2001:db8:1:2:3:4:5:6", "")
	require.NoError(t, err)
	assert.Equal(t, refusals.Caller{Scope: "2001:db8:1:2::/64"}, scope, "an address is refused as its scope")

	account, err := refusals.ParseCaller("", "01926C6E-7A4B-7C3D-8E9F-0A1B2C3D4E5F")
	require.NoError(t, err)
	assert.Equal(t, refusals.Caller{Account: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"}, account)

	_, err = refusals.ParseCaller("203.0.113.7", "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")
	require.ErrorIs(t, err, refusals.ErrNoCaller)
	_, err = refusals.ParseCaller("", "")
	require.ErrorIs(t, err, refusals.ErrNoCaller)
	_, err = refusals.ParseCaller("not an address", "")
	require.ErrorIs(t, err, refusals.ErrInvalidScope)
	_, err = refusals.ParseCaller("", "not-an-id")
	require.ErrorIs(t, err, refusals.ErrInvalidAccount)
}
