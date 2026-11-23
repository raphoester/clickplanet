package expire_flags_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/expire_flags_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAFlagWhoseTimeRanOutIsClosedAtItsEndAndTold(t *testing.T) {
	openedAt := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	store, outbox := catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	ending := catalog.OpenFlag("r-0000000001", "Rennes", openedAt)
	ends := ending.TakeSnapshot().EndsAt
	require.NoError(t, store.Save(t.Context(), ending, catalog.OpenFlag("r-0000000002", "Bavaria", openedAt.Add(time.Hour))))
	clock := cptime.NewFixedClock(ends.Add(time.Minute))
	useCase := expire_flags_usecase.New(clock, cpatomic.FakeTransactioner{}, store, outbox)

	closed, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, closed)

	kept, err := store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.Equal(t, catalog.Closed, kept.TakeSnapshot().State)
	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&flagsv1.FlagClosed{Flag: "r-0000000001", ClosedAt: timestamppb.New(ends)},
		outbox.Recorded()[0]), "%v", outbox.Recorded()[0])

	again, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Zero(t, again, "closed once")
}
