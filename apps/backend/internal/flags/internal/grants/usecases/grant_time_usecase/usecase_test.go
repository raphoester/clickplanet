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
	founder = catalog.AccountID{15: 1}
	other   = catalog.AccountID{15: 2}
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

// paid is the founder's payment, ordered a minute before.
func paid(t *testing.T, grant string, target grants.Target, period grants.Period) grants.Paid {
	t.Helper()
	return paidBy(grant, target, period, founder, now.Add(-time.Minute))
}

func paidBy(grant string, target grants.Target, period grants.Period, by catalog.AccountID, orderedAt time.Time) grants.Paid {
	return grants.Paid{Grant: grants.GrantID(grant), Target: target, Period: period, By: by, OrderedAt: orderedAt, At: now}
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

func TestThePaymentOfAFlagsHolderAddsItsPeriodToItsEnd(t *testing.T) {
	f := newFixture()
	require.NoError(t, f.flags.Save(t.Context(), catalog.HeldFlag("r-0000000001", "Rennes", founder, now)))

	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Year)))

	assert.Equal(t, now.Add(30*24*time.Hour).AddDate(1, 0, 0), f.flag(t, "r-0000000001").EndsAt)
	assert.Empty(t, f.outbox.Recorded())
}

// ended is Rennes, held by the founder, whose time ended two years ago.
func (f fixture) ended(t *testing.T) {
	t.Helper()
	flag, ok := catalog.HeldFlag("r-0000000001", "Rennes", founder, now.AddDate(-2, 0, 0)).Expired(now.AddDate(-2, 1, 0))
	require.True(t, ok)
	require.NoError(t, f.flags.Save(t.Context(), flag))
}

func TestTheHoldersPaymentOpensItsFlagAgainHoweverLongAgoItEnded(t *testing.T) {
	f := newFixture()
	f.ended(t)

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

func TestAnotherAccountsPaymentClaimsAFlagWhoseTimeEnded(t *testing.T) {
	f := newFixture()
	f.ended(t)

	require.NoError(t, f.grant.Execute(t.Context(),
		paidBy("stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Week, other, now.Add(-time.Minute))))

	flag := f.flag(t, "r-0000000001")
	assert.Equal(t, catalog.Open, flag.State)
	assert.Equal(t, other, flag.Holder, "the claim makes the payer its holder")
	assert.Equal(t, now.AddDate(0, 0, 7), flag.EndsAt)
	assert.Equal(t, catalog.Name("Rennes"), flag.Name, "the same flag, with no review")
	require.Len(t, f.outbox.Recorded(), 1)
}

func TestALateRenewalOfTheOldHolderIsRefusedOnceAnotherAccountClaimedTheFlag(t *testing.T) {
	f := newFixture()
	f.ended(t)
	f.submissions.Seed(opened(founder))
	require.NoError(t, f.grant.Execute(t.Context(),
		paidBy("stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Month, other, now.Add(-time.Minute))))

	err := f.grant.Execute(t.Context(),
		paidBy("stripe:in_9", submissionTarget(t), grants.Month, founder, now.AddDate(-2, 0, 0)))

	require.ErrorIs(t, err, grants.ErrRefused)
	require.ErrorIs(t, err, catalog.ErrHeld)
	assert.Equal(t, now.AddDate(0, 1, 0), f.flag(t, "r-0000000001").EndsAt, "the claim's month alone")
	assert.Equal(t, other, f.flag(t, "r-0000000001").Holder)
}

func TestALateRenewalOrderedBeforeTheFlagEndedClaimsNothing(t *testing.T) {
	f := newFixture()
	f.ended(t)
	f.submissions.Seed(opened(founder))
	require.NoError(t, f.grant.Execute(t.Context(),
		paidBy("stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Week, other, now.Add(-time.Minute))))
	f.clock.Advance(30 * 24 * time.Hour)
	expired, ok := f.flagged(t, "r-0000000001").Expired(f.clock.Now())
	require.True(t, ok)
	require.NoError(t, f.flags.Save(t.Context(), expired))

	err := f.grant.Execute(t.Context(),
		paidBy("stripe:in_9", submissionTarget(t), grants.Month, founder, now.AddDate(-2, 0, 0)))

	require.ErrorIs(t, err, catalog.ErrOrderedBefore, "the founder's plan was bought before the claim")
	assert.Equal(t, catalog.Closed, f.flag(t, "r-0000000001").State)
	assert.Equal(t, other, f.flag(t, "r-0000000001").Holder)
}

func TestARefundOfTheOldHoldersPaymentLeavesTheClaimsTime(t *testing.T) {
	f := newFixture()
	f.approve()
	require.NoError(t, f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", submissionTarget(t), grants.Week)))
	f.clock.Advance(8 * 24 * time.Hour)
	expired, ok := f.flagged(t, "r-0000000001").Expired(f.clock.Now())
	require.True(t, ok)
	require.NoError(t, f.flags.Save(t.Context(), expired))
	claim := paidBy("stripe:cs_2", flagTarget(t, "r-0000000001"), grants.Month, other, f.clock.Now())
	claim.At = f.clock.Now()
	require.NoError(t, f.grant.Execute(t.Context(), claim))

	require.NoError(t, f.takeBack.Execute(t.Context(), "stripe:cs_1"))

	flag := f.flag(t, "r-0000000001")
	assert.Equal(t, catalog.Open, flag.State, "the founder's week was spent before the claim")
	assert.Equal(t, f.clock.Now().AddDate(0, 1, 0), flag.EndsAt)
}

func TestAPaymentForARemovedFlagGrantsNothing(t *testing.T) {
	f := newFixture()
	removed, err := catalog.HeldFlag("r-0000000001", "Rennes", founder, now.Add(-40*24*time.Hour)).Removed("Broke the rules.", now.Add(-10*24*time.Hour))
	require.NoError(t, err)
	require.NoError(t, f.flags.Save(t.Context(), removed))

	err = f.grant.Execute(t.Context(), paid(t, "stripe:cs_1", flagTarget(t, "r-0000000001"), grants.Week))

	require.ErrorIs(t, err, grants.ErrRefused)
	require.ErrorIs(t, err, catalog.ErrRemoved)
	assert.Equal(t, catalog.Removed, f.flag(t, "r-0000000001").State)
	f.refused(t, "stripe:cs_1")
}

func opened(account catalog.AccountID) submissions.Snapshot {
	row := submissions.TestWaiting(id, account, now.AddDate(-2, 0, -1))
	row.State, row.Held, row.Flag = submissions.Opened, false, "r-0000000001"
	return row
}

func (f fixture) flagged(t *testing.T, id catalog.FlagID) catalog.Flag {
	t.Helper()
	flag, err := f.flags.Flag(t.Context(), id)
	require.NoError(t, err)
	return flag
}

// refused checks the payment bought nothing for good, and that the shop is told to refund it.
func (f fixture) refused(t *testing.T, grant grants.GrantID) {
	t.Helper()
	kept, err := f.grants.Grant(t.Context(), grant)
	require.NoError(t, err)
	assert.True(t, kept.TakenBackAlready(), "it never grants, its refund takes back nothing")
	_, granted := kept.GrantedFlag()
	assert.False(t, granted)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.TimeRefused{GrantId: grant.String(), RefusedAt: timestamppb.New(f.clock.Now())},
		f.outbox.Recorded()[0]))

	again := f.grant.Execute(t.Context(), grants.Paid{Grant: grant, Target: submissionTarget(t), Period: grants.Week})
	require.NoError(t, again, "told again, refused once")
	assert.Len(t, f.outbox.Recorded(), 1)
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
	f.refused(t, "stripe:cs_1")
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
