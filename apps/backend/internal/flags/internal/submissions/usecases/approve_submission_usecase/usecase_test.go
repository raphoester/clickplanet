package approve_submission_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/approve_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada = submissions.AccountID{15: 1}
	id  = submissions.SubmissionID{15: 1}
)

type fixture struct {
	store   *submissions.MemoryStore
	flags   *catalog.MemoryStore
	bucket  *pictures.MemoryBucket
	outbox  *cpatomic.FakeOutbox
	drawn   pictures.Drawn
	useCase *approve_submission_usecase.UseCase
}

func newFixture(t *testing.T, submission submissions.Snapshot) fixture {
	t.Helper()
	store, flags, bucket, outbox := submissions.NewMemoryStore(), catalog.NewMemoryStore(), pictures.NewMemoryBucket(),
		&cpatomic.FakeOutbox{}
	drawn, err := pictures.Redraw(pictures.TestPNG(300, 200))
	require.NoError(t, err)
	gallery := pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20})
	require.NoError(t, gallery.Hold(t.Context(), id.PictureKey(), drawn))
	submission.Picture = drawn.Picture.TakeSnapshot()
	store.Seed(submission)
	return fixture{
		store: store, flags: flags, bucket: bucket, outbox: outbox, drawn: drawn,
		useCase: approve_submission_usecase.New(cptime.NewFixedClock(now), gallery, cpatomic.FakeTransactioner{}, store,
			flags, outbox),
	}
}

func TestAnApprovedNewFlagIsPublicAndWaitsToBePaidFor(t *testing.T) {
	f := newFixture(t, submissions.TestWaiting(id, ada, now.Add(-time.Hour)))

	approved, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: id.String()})
	require.NoError(t, err)

	assert.Equal(t, submissions.Approved, approved.State)
	assert.Equal(t, now, approved.DecidedAt)
	assert.Equal(t, f.drawn.PNG, f.bucket.Published()[f.drawn.Picture.Key()], "public at its content's name")
	assert.Empty(t, f.bucket.HeldPictures(), "no longer held in private")
	assert.Equal(t, []submissions.Snapshot{approved}, f.store.Kept())
	assert.Empty(t, f.outbox.Recorded(), "no flag opens before it is paid for")
}

func TestAnApprovedChangeIsOnItsFlagForEveryPlayer(t *testing.T) {
	change := submissions.TestWaiting(id, ada, now.Add(-time.Hour))
	change.Kind, change.Flag, change.Name = submissions.Change, "r-0000000001", "Breizh"
	f := newFixture(t, change)
	require.NoError(t, f.flags.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", now.Add(-24*time.Hour))))

	_, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: id.String()})
	require.NoError(t, err)

	flag, err := f.flags.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Name("Breizh"), flag.TakeSnapshot().Name)
	assert.Equal(t, f.drawn.Picture.TakeSnapshot(), flag.TakeSnapshot().Picture)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagRenamed{
		Flag: "r-0000000001", Name: "Breizh", RenamedAt: timestamppb.New(now),
	}, f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestASubmissionDecidedIsNotApprovedAgain(t *testing.T) {
	rejected := submissions.TestWaiting(id, ada, now.Add(-time.Hour))
	rejected.State, rejected.Rejection = submissions.Rejected, submissions.Rejection{Reason: submissions.Hateful}
	f := newFixture(t, rejected)

	_, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: id.String()})

	require.ErrorIs(t, err, submissions.ErrDecided)
	assert.Empty(t, f.bucket.Published(), "a rejected picture never becomes public")
}

func TestAnUnknownSubmissionIsNotApproved(t *testing.T) {
	f := newFixture(t, submissions.TestWaiting(id, ada, now))

	_, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: submissions.SubmissionID{15: 2}.String()})
	require.ErrorIs(t, err, submissions.ErrUnknownSubmission)
	_, err = f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: "Rennes"})
	require.ErrorIs(t, err, submissions.ErrUnknownSubmission)
}

func TestANewFlagWithAnotherFlagsNameIsNotApproved(t *testing.T) {
	f := newFixture(t, submissions.TestWaiting(id, ada, now.Add(-time.Hour)))
	require.NoError(t, f.flags.Save(t.Context(), catalog.OpenFlag("r-0000000001", "rennes", now.Add(-24*time.Hour))))

	_, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: id.String()})

	require.ErrorIs(t, err, catalog.ErrNameTaken)
	assert.Empty(t, f.bucket.Published())
	assert.Equal(t, submissions.Waiting, f.store.Kept()[0].State)
}

func TestTheOperatorMovesTheCapitalBeforeTheApproval(t *testing.T) {
	f := newFixture(t, submissions.TestWaiting(id, ada, now.Add(-time.Hour)))

	approved, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{
		Submission: id.String(), Moved: true, Latitude: 47.218, Longitude: -1.5536,
	})
	require.NoError(t, err)

	assert.Equal(t, catalog.Pin{Latitude: 47.218, Longitude: -1.5536}, approved.Pin)
	assert.Equal(t, catalog.Pin{Latitude: 47.218, Longitude: -1.5536}, f.store.Kept()[0].Pin)

	_, err = f.useCase.Execute(t.Context(), approve_submission_usecase.In{Submission: id.String(), Moved: true, Latitude: 95})
	require.ErrorIs(t, err, catalog.ErrInvalidPin)
}

func TestAChangeApprovedMovesItsFlagsCapital(t *testing.T) {
	change := submissions.TestWaiting(id, ada, now.Add(-time.Hour))
	change.Kind, change.Flag, change.Name = submissions.Change, "r-0000000001", "Breizh"
	f := newFixture(t, change)
	require.NoError(t, f.flags.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", now.Add(-24*time.Hour))))

	_, err := f.useCase.Execute(t.Context(), approve_submission_usecase.In{
		Submission: id.String(), Moved: true, Latitude: 47.996, Longitude: -4.102,
	})
	require.NoError(t, err)

	flag, err := f.flags.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Pin{Latitude: 47.996, Longitude: -4.102}, flag.TakeSnapshot().Pin)
	require.Len(t, f.outbox.Recorded(), 2, "renamed, and moved for the next Capital Siege")
	assert.True(t, proto.Equal(&flagsv1.FlagMoved{
		Flag: "r-0000000001", Pin: &flagsv1.Pin{Latitude: 47.996, Longitude: -4.102}, MovedAt: timestamppb.New(now),
	}, f.outbox.Recorded()[1]), "%v", f.outbox.Recorded()[1])
}
