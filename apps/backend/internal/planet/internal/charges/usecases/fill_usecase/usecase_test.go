package fill_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/fill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice = charges.Holder("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

var at = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

type recordingStreams struct {
	told []charges.Gain
}

func (s *recordingStreams) PublishGranted(_ charges.Holder, gain charges.Gain) {
	s.told = append(s.told, gain)
}

func setUp(grants fill_usecase.Grants) (*fill_usecase.UseCase, *charges.MemoryStore, *recordingStreams) {
	hands := charges.NewMemoryStore()
	streams := &recordingStreams{}
	return fill_usecase.New(cpatomic.FakeTransactioner{}, grants, charges.NewCharges(hands, charges.TestRules), streams,
		cptime.NewFixedClock(at)), hands, streams
}

var fill = fill_usecase.In{
	Grant: charges.GrantID{15: 1}, Holder: alice, Kinds: []charges.Kind{charges.KindShields, charges.KindBomb},
}

func TestAFillTopsEachStackNamedAndIsToldOnceWithEveryKind(t *testing.T) {
	useCase, hands, streams := setUp(inmemory_grant_store.New(time.Minute))
	hands.Keep(alice, charges.Held{Shields: 4, Mines: 2})

	require.NoError(t, useCase.Execute(t.Context(), fill))

	assert.Equal(t, charges.Held{Shields: 30, Bombs: 1, Mines: 2}, hands.Kept(alice))
	require.Len(t, streams.told, 1, "one grant, one notice")
	assert.Equal(t, []charges.Kind{charges.KindBomb, charges.KindShields}, streams.told[0].Paid)
	assert.Equal(t, 26, streams.told[0].Count(charges.KindShields))
}

func TestAFillToldAgainFillsNothingAndTellsNothing(t *testing.T) {
	useCase, hands, streams := setUp(inmemory_grant_store.New(time.Minute))

	require.NoError(t, useCase.Execute(t.Context(), fill))
	hands.Keep(alice, charges.Held{})
	require.NoError(t, useCase.Execute(t.Context(), fill))

	assert.Equal(t, charges.Held{}, hands.Kept(alice))
	assert.Len(t, streams.told, 1)
}

type failingGrants struct{}

func (failingGrants) Record(context.Context, charges.GrantID, time.Time) (bool, error) {
	return false, errors.New("postgres is down")
}

func TestAFillWhoseGrantCannotBeKeptFillsNothing(t *testing.T) {
	useCase, hands, streams := setUp(failingGrants{})

	require.Error(t, useCase.Execute(t.Context(), fill))

	assert.Equal(t, charges.Held{}, hands.Kept(alice))
	assert.Empty(t, streams.told)
}
