package pay_signed_in_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/pay_signed_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type recordingPayroll struct {
	days []days.Day
	open bool
}

func (p *recordingPayroll) Pay(ctx context.Context, day days.Day, _ missions.AccountID) error {
	p.days = append(p.days, day)
	p.open = (&cpatomic.FakeOutbox{}).Record(ctx) == nil
	return nil
}

func TestASignInPaysTheDaysMissionsInATransaction(t *testing.T) {
	now := time.Date(2026, 10, 26, 22, 0, 0, 0, time.UTC)
	payroll := &recordingPayroll{}

	require.NoError(t, pay_signed_in_usecase.New(inmemory_day_store.New(days.TestingDayAt(now)), cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, payroll).
		Execute(t.Context(), missions.AccountID{15: 1}))

	assert.Equal(t, []days.Day{days.TestingDayAt(now)}, payroll.days, "the day the gamemaster told")
	assert.True(t, payroll.open)
}

func TestASignInWhenNoDayRunsPaysNothing(t *testing.T) {
	payroll := &recordingPayroll{}

	require.NoError(t, pay_signed_in_usecase.New(inmemory_day_store.New(), cptime.NewFixedClock(time.Now()), cpatomic.FakeTransactioner{}, payroll).
		Execute(t.Context(), missions.AccountID{15: 1}))

	assert.Empty(t, payroll.days)
}
