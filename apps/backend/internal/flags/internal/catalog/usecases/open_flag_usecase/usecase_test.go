package open_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/open_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	upload = pictures.UploadID{1}
)

type fixture struct {
	store   *catalog.MemoryStore
	bucket  *pictures.MemoryBucket
	outbox  *cpatomic.FakeOutbox
	useCase *open_flag_usecase.UseCase
}

func newFixture() fixture {
	store, bucket, outbox := catalog.NewMemoryStore(), pictures.NewMemoryBucket(), &cpatomic.FakeOutbox{}
	bucket.Upload(upload, pictures.TestPNG(300, 200))
	return fixture{store: store, bucket: bucket, outbox: outbox, useCase: open_flag_usecase.New(
		catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(now)),
		pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20}), cpatomic.FakeTransactioner{}, store, outbox,
	)}
}

func TestAFlagOpenedIsKeptWithItsPictureAndToldAndItsUploadGoes(t *testing.T) {
	f := newFixture()

	opened, err := f.useCase.Execute(t.Context(),
		open_flag_usecase.In{
			Name: " Rennes ", Upload: upload.String(), Latitude: 48.1173, Longitude: -1.6778, EndsAt: now.Add(7 * 24 * time.Hour),
		})
	require.NoError(t, err)

	assert.Equal(t, catalog.FlagID("r-0000000001"), opened.ID)
	assert.Equal(t, catalog.Name("Rennes"), opened.Name)
	assert.Equal(t, catalog.Open, opened.State)
	assert.Equal(t, catalog.Pin{Latitude: 48.1173, Longitude: -1.6778}, opened.Pin)
	assert.Equal(t, now.Add(7*24*time.Hour), opened.EndsAt)
	assert.Equal(t, [2]uint32{300, 200}, [2]uint32{opened.Picture.Width, opened.Picture.Height})
	assert.Contains(t, f.bucket.Published(), opened.Picture.Key(), "public")
	assert.Zero(t, f.bucket.Uploads(), "the file uploaded is gone once the flag is kept")
	kept, err := f.store.Flag(t.Context(), opened.ID)
	require.NoError(t, err)
	assert.Equal(t, opened, kept.TakeSnapshot())
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(
		&flagsv1.FlagOpened{Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(now)},
		f.outbox.Recorded()[0],
	), "%v", f.outbox.Recorded()[0])
}

func TestAFlagThatEndsBeforeNowIsNotOpenedAndItsUploadStays(t *testing.T) {
	f := newFixture()

	_, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "Rennes", Upload: upload.String(), EndsAt: now})

	require.ErrorIs(t, err, catalog.ErrEnded)
	assert.Equal(t, 1, f.bucket.Uploads(), "the operator may try again with the same file")
	assert.Empty(t, f.outbox.Recorded())
}

func TestAFlagWithNoNameNoPinOrNoPictureIsNotOpened(t *testing.T) {
	f := newFixture()
	ends := now.Add(time.Hour)

	_, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "  ", Upload: upload.String(), EndsAt: ends})
	require.ErrorIs(t, err, catalog.ErrInvalidName)
	_, err = f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "Rennes", Upload: pictures.UploadID{2}.String(), EndsAt: ends})
	require.ErrorIs(t, err, pictures.ErrUnknownUpload)
	_, err = f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "Rennes", Upload: upload.String(), Latitude: 91, EndsAt: ends})
	require.ErrorIs(t, err, catalog.ErrInvalidPin)
	f.bucket.Upload(upload, pictures.TestPNG(500, 200))
	_, err = f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "Rennes", Upload: upload.String(), EndsAt: ends})
	require.ErrorIs(t, err, pictures.ErrShape)

	flags, err := f.store.Flags(t.Context())
	require.NoError(t, err)
	assert.Empty(t, flags)
	assert.Empty(t, f.outbox.Recorded())
}

func TestAFlagIsNotOpenedUnderANameAnotherFlagHas(t *testing.T) {
	f := newFixture()
	require.NoError(t, f.store.Save(t.Context(), catalog.OpenFlag("r-0000000009", "Bavaria", now)))
	ends := now.Add(time.Hour)

	_, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "BAVARIA", Upload: upload.String(), EndsAt: ends})
	require.ErrorIs(t, err, catalog.ErrNameTaken)
	_, err = f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "germany", Upload: upload.String(), EndsAt: ends})
	require.ErrorIs(t, err, catalog.ErrNameTaken, "a flag the game ships with")

	assert.Equal(t, 1, f.bucket.Uploads(), "the operator may try again with another name")
	assert.Empty(t, f.outbox.Recorded())
}
