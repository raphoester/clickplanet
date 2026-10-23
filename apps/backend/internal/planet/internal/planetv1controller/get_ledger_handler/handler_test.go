package get_ledger_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/excerpt_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
)

var at = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

type stubUseCase struct {
	in  excerpt_usecase.In
	out excerpt_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in excerpt_usecase.In) (excerpt_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func TestTheWindowReachesTheUseCaseAndAnUnsetUntilIsLeftForItToFill(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := get_ledger_handler.New(useCase).GetLedger(t.Context(), connect.NewRequest(&planetv1.GetLedgerRequest{
		Since: timestamppb.New(at),
	}))
	require.NoError(t, err)

	assert.Equal(t, excerpt_usecase.In{Since: at}, useCase.in)
}

func TestAnInvalidWindowIsTheCallersMistake(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("wrapped: %w", excerpt_usecase.ErrInvalidWindow)}

	_, err := get_ledger_handler.New(useCase).GetLedger(t.Context(), connect.NewRequest(&planetv1.GetLedgerRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestEveryChangeGoesOutAsTheStreamSentItWithWhenAndWhoInOneString(t *testing.T) {
	useCase := &stubUseCase{out: excerpt_usecase.Out{
		Lines: []ledger.Line{
			{
				At: at.Add(1500 * time.Millisecond), By: ledger.Caller{Scope: "203.0.113.7"},
				Update: clicks.TileUpdate{Tile: 7, Value: "fr", Clicked: true},
			},
			{
				At: at.Add(time.Minute), By: ledger.Caller{Account: "acc"},
				Update: clicks.TileUpdate{Tile: 8, Value: "de", Previous: "fr"},
			},
			{
				At: at.Add(2 * time.Minute), By: ledger.Caller{Account: "acc"},
				Update: clicks.TileUpdate{Tile: 7, Value: "fr", Previous: "fr", Clicked: true, Shields: 2},
			},
		},
	}}

	res, err := get_ledger_handler.New(useCase).GetLedger(t.Context(),
		connect.NewRequest(&planetv1.GetLedgerRequest{Since: timestamppb.New(at)}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&planetv1.GetLedgerResponse{
		Changes: []*planetv1.LedgerChange{
			{
				At:     timestamppb.New(at.Add(1500 * time.Millisecond)),
				Update: &planetv1.TileUpdate{TileId: 7, CountryId: "fr", Clicked: true},
				By:     "scope:203.0.113.7",
			},
			{
				At:     timestamppb.New(at.Add(time.Minute)),
				Update: &planetv1.TileUpdate{TileId: 8, CountryId: "de", PreviousCountryId: "fr"},
				By:     "account:acc",
			},
			{
				At:     timestamppb.New(at.Add(2 * time.Minute)),
				Update: &planetv1.TileUpdate{TileId: 7, CountryId: "fr", PreviousCountryId: "fr", Clicked: true, Shields: 2},
				By:     "account:acc",
			},
		},
	}, res.Msg), res.Msg.String())
}
