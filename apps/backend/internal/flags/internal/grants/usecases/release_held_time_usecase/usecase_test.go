package release_held_time_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/grant_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/release_held_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	paidAt  = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	founder = submissions.AccountID{15: 1}
	id      = submissions.SubmissionID{15: 1}
)

type fixture struct {
	grants      *grants.MemoryStore
	submissions *submissions.MemoryStore
	flags       *catalog.MemoryStore
	outbox      *cpatomic.FakeOutbox
	clock       *cptime.FixedClock
	grant       *grant_time_usecase.UseCase
	release     *release_held_time_usecase.UseCase
}

func newFixture() fixture {
	f := fixture{
		grants: grants.NewMemoryStore(), submissions: submissions.NewMemoryStore(), flags: catalog.NewMemoryStore(),
		outbox: &cpatomic.FakeOutbox{}, clock: cptime.NewFixedClock(paidAt),
	}
	opening := catalog.NewOpening(&catalog.Sequence{}, f.clock)
	f.grant = grant_time_usecase.New(opening, f.clock, cpatomic.FakeTransactioner{},
		grant_time_usecase.Stores{Grants: f.grants, Submissions: f.submissions, Flags: f.flags}, f.outbox)
	f.release = release_held_time_usecase.New(opening, f.clock, cpatomic.FakeTransactioner{},
		release_held_time_usecase.Stores{Grants: f.grants, Submissions: f.submissions, Flags: f.flags}, f.outbox)
	f.submissions.Seed(submissions.TestWaiting(id, founder, paidAt.Add(-time.Hour)))
	return f
}

func (f fixture) pay(t *testing.T, grant string, period grants.Period) {
	t.Helper()
	target, err := grants.ParseTarget(id.String(), "")
	require.NoError(t, err)
	require.NoError(t, f.grant.Execute(t.Context(), grants.Paid{
		Grant: grants.GrantID(grant), Target: target, Period: period, At: f.clock.Now(),
	}))
}

// approve is what the operator's approval keeps: the release runs on the event it records.
func (f fixture) approve(t *testing.T) {
	t.Helper()
	approved := submissions.TestWaiting(id, founder, paidAt.Add(-time.Hour))
	approved.State, approved.Held, approved.DecidedAt = submissions.Approved, false, f.clock.Now()
	f.submissions.Seed(approved)
}

func (f fixture) flag(t *testing.T) catalog.Snapshot {
	t.Helper()
	flags, err := f.flags.Flags(t.Context())
	require.NoError(t, err)
	require.Len(t, flags, 1)
	return flags[0].TakeSnapshot()
}

func TestTheTimePaidBeforeTheReviewStartsAtTheApproval(t *testing.T) {
	f := newFixture()
	f.pay(t, "stripe:in_1", grants.Month)
	f.clock.Advance(3 * 24 * time.Hour)
	f.approve(t)

	require.NoError(t, f.release.Execute(t.Context(), id.String()))

	flag := f.flag(t)
	assert.Equal(t, catalog.Open, flag.State)
	approvedAt := paidAt.Add(3 * 24 * time.Hour)
	assert.Equal(t, approvedAt.AddDate(0, 1, 0), flag.EndsAt, "a month from its opening, not from the payment")
	require.Len(t, f.outbox.Recorded(), 1)
	assert.Equal(t, flag.ID.String(), f.outbox.Recorded()[0].(*flagsv1.FlagOpened).GetFlag())
	submission, err := f.submissions.Submission(t.Context(), id)
	require.NoError(t, err)
	opened, ok := submission.OpenedFlag()
	require.True(t, ok)
	assert.Equal(t, flag.ID, opened)
}

func TestEveryPaymentHeldAddsItsPeriodAndIsReleasedOnce(t *testing.T) {
	f := newFixture()
	f.pay(t, "stripe:in_1", grants.Month)
	f.pay(t, "stripe:in_2", grants.Week)
	f.approve(t)

	require.NoError(t, f.release.Execute(t.Context(), id.String()))
	require.NoError(t, f.release.Execute(t.Context(), id.String()))

	assert.Equal(t, paidAt.AddDate(0, 1, 7), f.flag(t).EndsAt)
	assert.Len(t, f.outbox.Recorded(), 1, "opened once")
	held, err := f.grants.Held(t.Context(), id)
	require.NoError(t, err)
	assert.Empty(t, held)
}

func TestAnApprovalWithNothingPaidOpensNothing(t *testing.T) {
	f := newFixture()
	f.approve(t)

	require.NoError(t, f.release.Execute(t.Context(), id.String()))

	flags, err := f.flags.Flags(t.Context())
	require.NoError(t, err)
	assert.Empty(t, flags, "it waits to be paid for")
	assert.Empty(t, f.outbox.Recorded())
}

func TestTimeHeldIsNotReleasedBeforeTheVerdict(t *testing.T) {
	f := newFixture()
	f.pay(t, "stripe:in_1", grants.Month)

	err := f.release.Execute(t.Context(), id.String())

	require.ErrorIs(t, err, grants.ErrStillWaiting)
	held, err := f.grants.Held(t.Context(), id)
	require.NoError(t, err)
	assert.Len(t, held, 1)
}
