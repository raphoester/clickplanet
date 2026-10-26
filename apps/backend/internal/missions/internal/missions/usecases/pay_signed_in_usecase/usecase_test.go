package pay_signed_in_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/pay_signed_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type recordingPayroll struct {
	days []missions.Day
	open bool
}

func (p *recordingPayroll) Pay(ctx context.Context, day missions.Day, _ missions.AccountID) error {
	p.days = append(p.days, day)
	p.open = (&cpatomic.FakeOutbox{}).Record(ctx) == nil
	return nil
}

func TestASignInPaysTheDaysMissionsInATransaction(t *testing.T) {
	calendar := missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	now := time.Date(2026, 10, 26, 22, 0, 0, 0, time.UTC)
	payroll := &recordingPayroll{}

	require.NoError(t, pay_signed_in_usecase.New(calendar, cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, payroll).
		Execute(t.Context(), missions.AccountID{15: 1}))

	assert.Equal(t, []missions.Day{calendar.DayOf(now)}, payroll.days, "after the cutoff, the next day")
	assert.True(t, payroll.open)
}
