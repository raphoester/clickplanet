package restate_flags_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/restate_flags_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
)

func TestEveryFlagIsToldAgainAsItIsNow(t *testing.T) {
	store, outbox := catalog.NewMemoryStore(), &cpatomic.FakeOutbox{}
	closed, err := catalog.OpenFlag("r-0000000002", "Bavaria", openedAt).Closed(closedAt)
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", openedAt), closed))

	require.NoError(t, restate_flags_usecase.New(cpatomic.FakeTransactioner{}, store, outbox).Execute(t.Context()))

	want := []proto.Message{
		&flagsv1.FlagOpened{Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(openedAt)},
		&flagsv1.FlagClosed{Flag: "r-0000000002", ClosedAt: timestamppb.New(closedAt)},
	}
	recorded := outbox.Recorded()
	require.Len(t, recorded, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], recorded[i]), "%v", recorded[i])
	}
}

func TestWithNoFlagOpenedNothingIsTold(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	require.NoError(t, restate_flags_usecase.New(cpatomic.FakeTransactioner{}, catalog.NewMemoryStore(), outbox).
		Execute(t.Context()))

	assert.Empty(t, outbox.Recorded())
}
