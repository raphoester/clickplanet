package renewals_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

var (
	now   = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	buyer = orders.AccountID{15: 1}
)

func told(state renewals.State, at time.Time) renewals.Told {
	kept := renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), at)
	return renewals.Told{ID: kept.ID, Order: kept.Order, State: state, PeriodEndsAt: kept.PeriodEndsAt, At: at}
}

func TestAWordOlderThanTheOneKeptChangesNothing(t *testing.T) {
	kept, changed, err := renewals.Heard(renewals.Renewal{}, false, told(renewals.Stopping, now))
	require.NoError(t, err)
	require.True(t, changed)

	heard, changed, err := renewals.Heard(kept, true, told(renewals.Renewing, now.Add(-time.Second)))
	require.NoError(t, err)

	assert.False(t, changed)
	assert.Equal(t, renewals.Stopping, heard.TakeSnapshot().State)
}

func TestTheBuyerAloneStopsItsRenewal(t *testing.T) {
	renewal, err := renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now).Restore()
	require.NoError(t, err)

	_, _, err = renewal.StoppedBy(orders.AccountID{15: 2}, now)
	require.ErrorIs(t, err, renewals.ErrNotYours)

	stopped, changed, err := renewal.StoppedBy(buyer, now)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, renewals.Stopping, stopped.TakeSnapshot().State)

	_, changed, err = stopped.StoppedBy(buyer, now.Add(time.Minute))
	require.NoError(t, err)
	assert.False(t, changed, "a renewal stopping stays so")
}

func TestAnEndedRenewalIsNotStopped(t *testing.T) {
	ended := renewals.TestRenewing("sub_1", buyer, now, now)
	ended.State = renewals.Ended
	renewal, err := ended.Restore()
	require.NoError(t, err)

	_, _, err = renewal.StoppedBy(buyer, now)

	require.ErrorIs(t, err, renewals.ErrEnded)
}
