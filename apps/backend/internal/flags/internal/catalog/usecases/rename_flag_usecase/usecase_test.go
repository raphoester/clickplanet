package rename_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/rename_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	now      = openedAt.Add(time.Hour)
)

type fixture struct {
	store   *catalog.MemoryStore
	outbox  *cpatomic.FakeOutbox
	useCase *rename_flag_usecase.UseCase
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store, outbox := catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	require.NoError(t, store.Save(t.Context(),
		catalog.OpenFlag("r-0000000001", "Rennes", openedAt), catalog.OpenFlag("r-0000000002", "Bavaria", openedAt)))
	return fixture{store: store, outbox: outbox, useCase: rename_flag_usecase.New(
		cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, store, outbox)}
}

func TestAFlagRenamedIsKeptUnderItsNewNameAndTold(t *testing.T) {
	f := newFixture(t)

	renamed, err := f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "r-0000000001", Name: " Roazhon "})
	require.NoError(t, err)

	assert.Equal(t, catalog.Name("Roazhon"), renamed.Name)
	kept, err := f.store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, renamed, kept.TakeSnapshot())
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagRenamed{Flag: "r-0000000001", Name: "Roazhon", RenamedAt: timestamppb.New(now)},
		f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestAFlagTakesNoNameAnotherFlagHas(t *testing.T) {
	f := newFixture(t)

	_, err := f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "r-0000000001", Name: "bavaria"})
	require.ErrorIs(t, err, catalog.ErrNameTaken)
	_, err = f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "r-0000000001", Name: "Brittany"})
	require.ErrorIs(t, err, catalog.ErrNameTaken, "a flag the game ships with")
	_, err = f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "r-0000000001", Name: "RENNES"})
	require.NoError(t, err, "its own name in another case")
	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestOnlyAFlagOpenedSinceTheGameShippedIsRenamed(t *testing.T) {
	f := newFixture(t)

	_, err := f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "r-0000000009", Name: "Savoy"})
	require.ErrorIs(t, err, catalog.ErrUnknownFlag)
	_, err = f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "fr", Name: "Savoy"})
	require.ErrorIs(t, err, catalog.ErrInvalidFlagID)
	_, err = f.useCase.Execute(t.Context(), rename_flag_usecase.In{Flag: "r-0000000001", Name: " "})
	require.ErrorIs(t, err, catalog.ErrInvalidName)
	assert.Empty(t, f.outbox.Recorded())
}
