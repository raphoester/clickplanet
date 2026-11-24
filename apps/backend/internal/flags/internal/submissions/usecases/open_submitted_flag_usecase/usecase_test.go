package open_submitted_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/open_submitted_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	id  = submissions.SubmissionID{15: 1}
)

func newUseCase(store *submissions.MemoryStore, flags *catalog.MemoryStore, outbox *cpatomic.FakeOutbox) *open_submitted_flag_usecase.UseCase {
	return open_submitted_flag_usecase.New(catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(now)),
		cpatomic.FakeTransactioner{}, store, flags, outbox)
}

func TestAnApprovedSubmissionOpensItsFlagForEveryPlayerAtItsPin(t *testing.T) {
	store, flags, outbox := submissions.NewMemoryStore(), catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	approved := submissions.TestWaiting(id, submissions.AccountID{15: 1}, now.Add(-time.Hour))
	approved.State, approved.Held = submissions.Approved, false
	store.Seed(approved)

	flag, err := newUseCase(store, flags, outbox).Execute(t.Context(),
		open_submitted_flag_usecase.In{Submission: id.String(), EndsAt: now.Add(7 * 24 * time.Hour)})
	require.NoError(t, err)

	assert.Equal(t, catalog.Snapshot{
		ID: "r-0000000001", Name: "Rennes", Picture: catalog.TestPicture, Pin: catalog.TestPin, State: catalog.Open,
		DecidedAt: now, EndsAt: now.Add(7 * 24 * time.Hour),
	}, flag)
	kept, err := flags.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, flag, kept.TakeSnapshot())
	assert.Equal(t, submissions.Opened, store.Kept()[0].State)
	assert.Equal(t, catalog.FlagID("r-0000000001"), store.Kept()[0].Flag)
	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagOpened{
		Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(now),
	}, outbox.Recorded()[0]), "%v", outbox.Recorded()[0])
}

func TestAWaitingSubmissionOpensNoFlag(t *testing.T) {
	store, flags, outbox := submissions.NewMemoryStore(), catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	store.Seed(submissions.TestWaiting(id, submissions.AccountID{15: 1}, now))

	_, err := newUseCase(store, flags, outbox).Execute(t.Context(),
		open_submitted_flag_usecase.In{Submission: id.String(), EndsAt: now.Add(time.Hour)})

	require.ErrorIs(t, err, submissions.ErrNotApproved)
	all, err := flags.Flags(t.Context())
	require.NoError(t, err)
	assert.Empty(t, all)
	assert.Empty(t, outbox.Recorded())
}
