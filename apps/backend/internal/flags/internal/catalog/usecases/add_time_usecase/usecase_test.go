package add_time_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/add_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(30 * 24 * time.Hour)
)

type fixture struct {
	store   *catalog.MemoryStore
	outbox  *cpatomic.FakeOutbox
	clock   *cptime.FixedClock
	useCase *add_time_usecase.UseCase
}

func newFixture(t *testing.T, flag catalog.Flag) fixture {
	t.Helper()
	store, outbox, clock := catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}, cptime.NewFixedClock(openedAt)
	require.NoError(t, store.Save(t.Context(), flag))
	return fixture{
		store: store, outbox: outbox, clock: clock,
		useCase: add_time_usecase.New(clock, cpatomic.FakeTransactioner{}, store, outbox),
	}
}

func TestTimeAddedToAnOpenFlagTellsNothing(t *testing.T) {
	f := newFixture(t, catalog.OpenFlag("r-0000000001", "Rennes", openedAt))

	added, err := f.useCase.Execute(t.Context(), add_time_usecase.In{Flag: "r-0000000001", Days: 7})
	require.NoError(t, err)

	assert.Equal(t, openedAt.Add(37*24*time.Hour), added.EndsAt)
	assert.Empty(t, f.outbox.Recorded(), "the others know it open already")
}

func TestAFlagClosedComesBackOpenedAgainWithinThirtyDays(t *testing.T) {
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed("Broke the rules.", closedAt)
	require.NoError(t, err)
	f := newFixture(t, closed)
	back := closedAt.Add(10 * 24 * time.Hour)
	f.clock.Advance(back.Sub(openedAt))

	added, err := f.useCase.Execute(t.Context(), add_time_usecase.In{Flag: "r-0000000001", Days: 30})
	require.NoError(t, err)

	assert.Equal(t, catalog.Open, added.State)
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(
		&flagsv1.FlagOpened{Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(back)},
		f.outbox.Recorded()[0]), "%v", f.outbox.Recorded()[0])
}

func TestAFlagClosedThirtyDaysAgoDoesNotComeBack(t *testing.T) {
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed("Broke the rules.", closedAt)
	require.NoError(t, err)
	f := newFixture(t, closed)
	f.clock.Advance(closedAt.Add(30 * 24 * time.Hour).Sub(openedAt))

	_, err = f.useCase.Execute(t.Context(), add_time_usecase.In{Flag: "r-0000000001", Days: 30})

	require.ErrorIs(t, err, catalog.ErrTooLate)
	kept, err := f.store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Closed, kept.TakeSnapshot().State)
	assert.Empty(t, f.outbox.Recorded())
}

func TestDaysAndCodesAreChecked(t *testing.T) {
	f := newFixture(t, catalog.OpenFlag("r-0000000001", "Rennes", openedAt))

	_, err := f.useCase.Execute(t.Context(), add_time_usecase.In{Flag: "r-0000000001", Days: 0})
	require.ErrorIs(t, err, catalog.ErrInvalidDays)
	_, err = f.useCase.Execute(t.Context(), add_time_usecase.In{Flag: "fr", Days: 7})
	require.ErrorIs(t, err, catalog.ErrInvalidFlagID)
	_, err = f.useCase.Execute(t.Context(), add_time_usecase.In{Flag: "r-0000000002", Days: 7})
	require.ErrorIs(t, err, catalog.ErrUnknownFlag)
}
