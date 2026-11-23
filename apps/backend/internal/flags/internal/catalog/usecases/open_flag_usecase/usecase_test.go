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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

type fixture struct {
	store   *catalog.MemoryStore
	outbox  *cpatomic.FakeOutbox
	useCase *open_flag_usecase.UseCase
}

func newFixture() fixture {
	store, outbox := catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	return fixture{store: store, outbox: outbox, useCase: open_flag_usecase.New(
		&catalog.Sequence{}, cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, store, outbox)}
}

func TestAFlagOpenedIsKeptAndTold(t *testing.T) {
	f := newFixture()

	out, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: " Rennes "})
	require.NoError(t, err)

	assert.Equal(t, catalog.FlagID("r-0000000001"), out.Flag)
	kept, err := f.store.Flag(t.Context(), out.Flag)
	require.NoError(t, err)
	assert.Equal(t, catalog.Snapshot{ID: out.Flag, Name: "Rennes", State: catalog.Open, DecidedAt: now},
		kept.TakeSnapshot())
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(
		&flagsv1.FlagOpened{Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(now)},
		f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestEachFlagOpenedHasAnIDOfItsOwn(t *testing.T) {
	f := newFixture()

	first, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "Rennes"})
	require.NoError(t, err)
	second, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "Rennes"})
	require.NoError(t, err)

	assert.NotEqual(t, first.Flag, second.Flag, "a name is not an id")
}

func TestAFlagWithNoNameIsNotOpened(t *testing.T) {
	f := newFixture()

	_, err := f.useCase.Execute(t.Context(), open_flag_usecase.In{Name: "  "})
	require.ErrorIs(t, err, catalog.ErrInvalidName)

	flags, err := f.store.Flags(t.Context())
	require.NoError(t, err)
	assert.Empty(t, flags)
	assert.Empty(t, f.outbox.Recorded())
}
