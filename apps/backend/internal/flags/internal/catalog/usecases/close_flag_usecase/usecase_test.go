package close_flag_usecase_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/close_flag_usecase"
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
	useCase *close_flag_usecase.UseCase
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store, outbox := catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	require.NoError(t, store.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", openedAt)))
	return fixture{store: store, outbox: outbox, useCase: close_flag_usecase.New(
		cptime.NewFixedClock(now), cpatomic.FakeTransactioner{}, store, outbox)}
}

func TestAFlagClosedIsKeptClosedAndTold(t *testing.T) {
	f := newFixture(t)

	answered, err := f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000001", Note: " Broke the rules. "})
	require.NoError(t, err)

	kept, err := f.store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Closed, kept.TakeSnapshot().State, "closed, never deleted")
	assert.Equal(t, now, kept.TakeSnapshot().DecidedAt)
	assert.Equal(t, kept.TakeSnapshot(), answered)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagClosed{Flag: "r-0000000001", ClosedAt: timestamppb.New(now)},
		f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestAFlagClosedAlreadyIsNotClosedAgain(t *testing.T) {
	f := newFixture(t)
	_, err := f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000001", Note: " Broke the rules. "})
	require.NoError(t, err)

	_, err = f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000001", Note: " Broke the rules. "})

	require.ErrorIs(t, err, catalog.ErrClosed)
	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestOnlyAFlagOpenedSinceTheGameShippedCanBeClosed(t *testing.T) {
	f := newFixture(t)

	_, err := f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000002", Note: " Broke the rules. "})
	require.ErrorIs(t, err, catalog.ErrUnknownFlag)
	_, err = f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "fr", Note: " Broke the rules. "})
	require.ErrorIs(t, err, catalog.ErrInvalidFlagID)
	assert.Empty(t, f.outbox.Recorded())
}

func TestAFlagClosedKeepsTheNoteItsFounderReads(t *testing.T) {
	f := newFixture(t)

	_, err := f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000001", Note: " Broke the rules. "})
	require.NoError(t, err)

	kept, err := f.store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Note("Broke the rules."), kept.TakeSnapshot().Note)
}

func TestAFlagIsNotClosedWithNoNote(t *testing.T) {
	f := newFixture(t)

	_, err := f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000001", Note: "  "})
	require.ErrorIs(t, err, catalog.ErrInvalidNote)
	_, err = f.useCase.Execute(t.Context(), close_flag_usecase.In{Flag: "r-0000000001", Note: strings.Repeat("é", 501)})
	require.ErrorIs(t, err, catalog.ErrInvalidNote)

	kept, err := f.store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Open, kept.TakeSnapshot().State)
	assert.Empty(t, f.outbox.Recorded())
}
