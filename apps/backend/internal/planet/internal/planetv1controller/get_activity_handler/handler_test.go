package get_activity_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/activity_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_activity_handler"
)

var at = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

type stubUseCase struct {
	in  activity_usecase.In
	out activity_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in activity_usecase.In) (activity_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func TestTheSpanReachesTheUseCaseAndAnUnsetOneIsLeftForItToFill(t *testing.T) {
	useCase := &stubUseCase{}
	handler := get_activity_handler.New(useCase, 100)

	_, err := handler.GetActivity(t.Context(), connect.NewRequest(&planetv1.GetActivityRequest{
		Span: durationpb.New(6 * time.Hour),
	}))
	require.NoError(t, err)
	assert.Equal(t, activity_usecase.In{Span: 6 * time.Hour}, useCase.in)

	_, err = handler.GetActivity(t.Context(), connect.NewRequest(&planetv1.GetActivityRequest{}))
	require.NoError(t, err)
	assert.Equal(t, activity_usecase.In{}, useCase.in)
}

func TestAnInvalidSpanIsTheCallersMistake(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("wrapped: %w", ledger.ErrInvalidSpan)}

	_, err := get_activity_handler.New(useCase, 100).GetActivity(t.Context(),
		connect.NewRequest(&planetv1.GetActivityRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestTheActivityGoesOutWithEveryCountryThatHoldsOrMovedMostHeldFirst(t *testing.T) {
	useCase := &stubUseCase{out: activity_usecase.Out{
		Activity: ledger.Summary{
			Since: at, Until: at.Add(30 * time.Minute), Step: 15 * time.Minute, Players: 2,
			Slots: []ledger.Slot{{Start: at, Clicks: 4, Tiles: 5, Players: 2}, {Start: at.Add(15 * time.Minute)}},
			Moves: map[string]ledger.Moves{"fr": {Won: 5}, "pl": {Lost: 3}, "de": {Lost: 2}},
		},
		Land:     []byte{1, 2},
		Heat:     []uint32{5, 0},
		Held:     map[string]uint32{"fr": 40, "de": 60},
		Audience: bonuses.Audience{Streams: 7, Entrants: 5},
	}}

	res, err := get_activity_handler.New(useCase, 100).GetActivity(t.Context(),
		connect.NewRequest(&planetv1.GetActivityRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&planetv1.GetActivityResponse{
		Since:   timestamppb.New(at),
		Until:   timestamppb.New(at.Add(30 * time.Minute)),
		Step:    durationpb.New(15 * time.Minute),
		Slots:   []*planetv1.ActivitySlot{{Clicks: 4, Tiles: 5, Players: 2}, {}},
		Players: 2,
		Heatmap: &planetv1.ActivityHeatmap{
			Columns: clicks.GridColumns, Rows: clicks.GridRows, Land: []byte{1, 2}, Changes: []uint32{5, 0},
		},
		Countries: []*planetv1.CountryActivity{
			{CountryId: "de", Held: 60, Lost: 2},
			{CountryId: "fr", Held: 40, Won: 5},
			{CountryId: "pl", Lost: 3},
		},
		Streams:  7,
		Watchers: 5,
		MapTiles: 100,
	}, res.Msg), res.Msg.String())
}
