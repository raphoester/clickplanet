package grant_time_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/grant_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/take_back_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
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
	takeBack    *take_back_time_usecase.UseCase
}

func newFixture() fixture {
	f := fixture{
		grants: grants.NewMemoryStore(), submissions: submissions.NewMemoryStore(), flags: catalog.NewMemoryStore(),
		outbox: &cpatomic.FakeOutbox{}, clock: cptime.NewFixedClock(now),
	}
	f.grant = grant_time_usecase.New(catalog.NewOpening(&catalog.Sequence{}, f.clock), f.clock,
		cpatomic.FakeTransactioner{},
		grant_time_usecase.Stores{Grants: f.grants, Submissions: f.submissions, Flags: f.flags}, f.outbox)
	f.takeBack = take_back_time_usecase.New(f.clock, cpatomic.FakeTransactioner{}, f.grants, f.flags, f.outbox)
	return f
}

func (f fixture) approve() {
	approved := submissions.TestWaiting(id, founder, now.Add(-time.Hour))
	approved.State, approved.Held = submissions.Approved, false
	f.submissions.Seed(approved)
}

func paid(t *testing.T, grant string, target grants.Target, period grants.Period) grants.Paid {
	t.Helper()
	return grants.Paid{Grant: grants.GrantID(grant), Target: target, Period: period, At: now}
}

func submissionTarget(t *testing.T) grants.Target {
	t.Helper()
	target, err := grants.ParseTarget(id.String(), "")
	require.NoError(t, err)
	return target
}

func flagTarget(t *testing.T, flag catalog.FlagID) grants.Target {
	t.Helper()
	target, err := grants.ParseTarget("", flag.String())
	require.NoError(t, err)
	return target
}

func (f fixture) flag(t *testing.T, id catalog.FlagID) catalog.Snapshot {
	t.Helper()
	flag, err := f.flags.Flag(t.Context(), id)
	require.NoError(t, err)
	return flag.TakeSnapshot()
}

func TestTheFirstPaymentOpensTheApprovedSubmissionsFlagForItsPeriod(t *testing.T) {
	f := newFixture()
	f.approve()

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))

	flag := f.flag(t, "r-0000000001")
	assert.Equal(t, catalog.Open, flag.State)
	assert.Equal(t, now.AddDate(0, 0, 7), flag.EndsAt)
	assert.Equal(t, submissions.Opened, f.submissions.Kept()[0].State)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagOpened{
		Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(now),
		Pin: &flagsv1.Pin{Latitude: 48.1173, Longitude: -1.6778},
	}, f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestAPaymentToldAgainGrantsOnce(t *testing.T) {
	f := newFixture()
	f.approve()

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Month)))
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Month)))

	assert.Equal(t, now.AddDate(0, 1, 0), f.flag(t, "r-0000000001").EndsAt)
	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestALaterPaymentForTheSubmissionAddsItsPeriodToItsFlag(t *testing.T) {
	f := newFixture()
	f.approve()
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:in_1", submissionTarget(t), grants.Month)))
	f.clock.Advance(24 * time.Hour)

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:in_2", submissionTarget(t), grants.Month)))

	assert.Equal(t, now.AddDate(0, 2, 0), f.flag(t, "r-0000000001").EndsAt, "a renewal adds to the end, not to now")
	assert.Len(t, f.outbox.Recorded(), 1, "the others know it open already")
}

func TestAPaymentForAFlagAddsItsPeriodToItsEnd(t *testing.T) {
	f := newFixture()
	require.NoError(t, f.flags.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", now)))

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Year)))

	assert.Equal(t, now.Add(30*24*time.Hour).AddDate(1, 0, 0), f.flag(t, "r-0000000001").EndsAt)
	assert.Empty(t, f.outbox.Recorded())
}

func TestAPaymentWithinThirtyDaysOpensAClosedFlagAgain(t *testing.T) {
	f := newFixture()
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", now.Add(-40*24*time.Hour)).Closed("Broke the rules.", now.Add(-10*24*time.Hour))
	require.NoError(t, err)
	require.NoError(t, f.flags.Save(t.Context(), closed))

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Week)))

	flag := f.flag(t, "r-0000000001")
	assert.Equal(t, catalog.Open, flag.State)
	assert.Equal(t, now.AddDate(0, 0, 7), flag.EndsAt)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagOpened{
		Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(now),
		Pin: &flagsv1.Pin{Latitude: 48.1173, Longitude: -1.6778},
	}, f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestAPaymentPastTheThirtyDaysGrantsNothing(t *testing.T) {
	f := newFixture()
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", now.Add(-90*24*time.Hour)).Closed("Broke the rules.", now.Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.NoError(t, f.flags.Save(t.Context(), closed))

	err = f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Week))

	require.ErrorIs(t, err, catalog.ErrTooLate)
	assert.Equal(t, catalog.Closed, f.flag(t, "r-0000000001").State)
	grant, err := f.grants.Grant(t.Context(), "stripe:cs_1")
	require.NoError(t, err)
	assert.False(t, grant.Heard(), "nothing was granted")
}

func TestAPaymentForASubmissionThatWaitsHoldsItsTimeAndOpensNothing(t *testing.T) {
	f := newFixture()
	f.submissions.Seed(submissions.TestWaiting(id, founder, now))

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))

	assert.Empty(t, f.outbox.Recorded())
	held, err := f.grants.Held(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, held, 1, "told twice, held once")
	assert.Equal(t, grants.Snapshot{ID: "stripe:cs_1", Held: true, Submission: id, Period: grants.Week, HeldAt: now},
		held[0].TakeSnapshot())
}

func TestARefundBeforeTheReviewTakesTheHeldTimeBack(t *testing.T) {
	f := newFixture()
	f.submissions.Seed(submissions.TestWaiting(id, founder, now))
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))

	require.NoError(t, f.takeBack.Execute(t.Context(), "stripe:cs_1"))

	held, err := f.grants.Held(t.Context(), id)
	require.NoError(t, err)
	assert.Empty(t, held)
	assert.Empty(t, f.outbox.Recorded())
}

func TestAPaymentForARejectedSubmissionGrantsNothing(t *testing.T) {
	f := newFixture()
	rejection, err := submissions.ParseRejection(submissions.NotAPlace, "")
	require.NoError(t, err)
	rejected := submissions.TestWaiting(id, founder, now)
	rejected.State, rejected.Rejection = submissions.Rejected, rejection
	f.submissions.Seed(rejected)

	err = f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week))

	require.ErrorIs(t, err, submissions.ErrNotApproved)
	grant, err := f.grants.Grant(t.Context(), "stripe:cs_1")
	require.NoError(t, err)
	assert.False(t, grant.Heard())
}

func TestARefundTakesBackThePaymentsTimeAndClosesTheFlagItOpened(t *testing.T) {
	f := newFixture()
	f.approve()
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))
	f.clock.Advance(time.Hour)

	require.NoError(t, f.takeBack.Execute(t.Context(), "stripe:cs_1"))

	flag := f.flag(t, "r-0000000001")
	assert.Equal(t, catalog.Closed, flag.State)
	assert.Equal(t, now.Add(time.Hour), flag.DecidedAt, "closed now, never before its opening")
	require.Len(t, f.outbox.Recorded(), 2)
	assert.True(t, proto.Equal(&flagsv1.FlagClosed{
		Flag: "r-0000000001", ClosedAt: timestamppb.New(now.Add(time.Hour)),
	}, f.outbox.Recorded()[1]), "%v", f.outbox.Recorded()[1])
}

func TestARefundOfARenewalEndsTheFlagAPeriodSooner(t *testing.T) {
	f := newFixture()
	f.approve()
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:in_1", submissionTarget(t), grants.Month)))
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:in_2", submissionTarget(t), grants.Month)))

	require.NoError(t, f.takeBack.Execute(t.Context(), "stripe:in_2"))
	require.NoError(t, f.takeBack.Execute(t.Context(), "stripe:in_2"))

	flag := f.flag(t, "r-0000000001")
	assert.Equal(t, catalog.Open, flag.State)
	assert.Equal(t, now.AddDate(0, 1, 0), flag.EndsAt, "taken back once")
}

func TestAPaymentTakenBackBeforeItsGrantGrantsNothing(t *testing.T) {
	f := newFixture()
	f.approve()

	require.NoError(t, f.takeBack.Execute(t.Context(), "stripe:cs_1"))
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))

	assert.Equal(t, submissions.Approved, f.submissions.Kept()[0].State)
	assert.Empty(t, f.outbox.Recorded())
}
