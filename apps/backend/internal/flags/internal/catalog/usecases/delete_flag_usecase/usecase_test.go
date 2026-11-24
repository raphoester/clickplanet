package delete_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/delete_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
	now      = closedAt.Add(time.Hour)
)

type fixture struct {
	store   *catalog.MemoryStore
	bucket  *pictures.MemoryBucket
	outbox  *cpatomic.FakeOutbox
	useCase *delete_flag_usecase.UseCase
}

func newFixture(t *testing.T, flags ...catalog.Flag) fixture {
	t.Helper()
	store, bucket, outbox := catalog.NewMemoryStore(), pictures.NewMemoryBucket(), &cpatomic.FakeOutbox{}
	require.NoError(t, store.Save(t.Context(), flags...))
	bucket.Show(catalog.TestPicture)
	return fixture{store: store, bucket: bucket, outbox: outbox, useCase: delete_flag_usecase.New(
		cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, store, outbox,
		pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20}))}
}

func closed(t *testing.T, id catalog.FlagID, name catalog.Name) catalog.Flag {
	t.Helper()
	flag, err := catalog.OpenFlag(id, name, openedAt).Closed(closedAt)
	require.NoError(t, err)
	return flag
}

func TestAClosedFlagNobodyPlayedIsGoneToldAndItsPictureRemoved(t *testing.T) {
	f := newFixture(t, closed(t, "r-0000000001", "Rennes"))

	require.NoError(t, f.useCase.Execute(t.Context(), delete_flag_usecase.In{Flag: "r-0000000001"}))

	_, err := f.store.Flag(t.Context(), "r-0000000001")
	require.ErrorIs(t, err, catalog.ErrUnknownFlag)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagDeleted{Flag: "r-0000000001", DeletedAt: timestamppb.New(now)},
		f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
	assert.NotContains(t, f.bucket.Published(), catalog.TestPicture.Key())
}

func TestAPictureAnotherFlagWearsStays(t *testing.T) {
	f := newFixture(t, closed(t, "r-0000000001", "Rennes"), catalog.OpenFlag("r-0000000002", "Bavaria", openedAt))

	require.NoError(t, f.useCase.Execute(t.Context(), delete_flag_usecase.In{Flag: "r-0000000001"}))

	assert.Contains(t, f.bucket.Published(), catalog.TestPicture.Key())
}

func TestAFlagOpenOrPlayedIsNeverDeleted(t *testing.T) {
	played, _ := closed(t, "r-0000000002", "Bavaria").Played()
	f := newFixture(t, catalog.OpenFlag("r-0000000001", "Rennes", openedAt), played)

	require.ErrorIs(t, f.useCase.Execute(t.Context(), delete_flag_usecase.In{Flag: "r-0000000001"}), catalog.ErrOpen)
	require.ErrorIs(t, f.useCase.Execute(t.Context(), delete_flag_usecase.In{Flag: "r-0000000002"}), catalog.ErrPlayed)
	require.ErrorIs(t, f.useCase.Execute(t.Context(), delete_flag_usecase.In{Flag: "r-0000000009"}), catalog.ErrUnknownFlag)

	flags, err := f.store.Flags(t.Context())
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.Empty(t, f.outbox.Recorded())
	assert.Contains(t, f.bucket.Published(), catalog.TestPicture.Key())
}
