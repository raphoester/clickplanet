package activity_get_map_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase/activity_get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubUseCase struct {
	batch clicks.DenseBatch
	err   error
}

func (s stubUseCase) Execute(context.Context, get_map_usecase.In) (clicks.DenseBatch, error) {
	return s.batch, s.err
}

type board uint32

func (b board) MaxIndex() uint32 { return uint32(b) }

var now = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func read(t *testing.T, inner stubUseCase, in get_map_usecase.In) (activity.Event, clicks.DenseBatch, error) {
	t.Helper()

	recorded := &activity.Recorded{}
	batch, err := activity_get_map.New(inner, recorded, board(262119), cptime.NewFixedClock(now)).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), in)

	require.Len(t, recorded.Events(), 1)
	return recorded.Events()[0], batch, err
}

func TestAReadIsRecordedAsAsked(t *testing.T) {
	inner := stubUseCase{batch: clicks.DenseBatch{Start: 1, Codes: []string{"", "fr"}, Tiles: make([]byte, 20)}}

	event, batch, err := read(t, inner, get_map_usecase.In{Start: 1, End: 10001})

	require.NoError(t, err)
	assert.Equal(t, inner.batch, batch, "the batch is the inner one, untouched")
	assert.Equal(t, activity.Event{
		At: now, Kind: activity.KindMap, Caller: activity.Caller{Scope: "2001:db8::/64", Account: cpsession.NoAccount},
		Start: 1, End: 10001, Outcome: activity.OutcomeAccepted,
	}, event)
}

func TestAReadOffTheMapSaysSo(t *testing.T) {
	event, _, err := read(t, stubUseCase{}, get_map_usecase.In{Start: 0, End: 10000})

	require.NoError(t, err)
	assert.True(t, event.OffMap)
}

func TestARefusedReadIsRecordedToo(t *testing.T) {
	event, _, err := read(t, stubUseCase{err: clicks.ErrInvalidTileRange}, get_map_usecase.In{Start: 9, End: 3})

	require.ErrorIs(t, err, clicks.ErrInvalidTileRange, "still the sentinel the handler maps")
	assert.Equal(t, activity.OutcomeInvalid, event.Outcome)
}

func TestAFaultOfThisServerIsAFailure(t *testing.T) {
	cause := errors.New("disk on fire")

	event, _, err := read(t, stubUseCase{err: cause}, get_map_usecase.In{Start: 1, End: 2})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, activity.OutcomeFailed, event.Outcome)
}
