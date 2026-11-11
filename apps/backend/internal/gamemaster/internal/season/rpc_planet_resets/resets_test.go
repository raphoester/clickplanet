package rpc_planet_resets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/rpc_planet_resets"
)

type planet struct {
	asked []*planetv1.ResetMapRequest
	err   error
}

func (p *planet) ResetMap(_ context.Context, req *connect.Request[planetv1.ResetMapRequest],
) (*connect.Response[planetv1.ResetMapResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(&planetv1.ResetMapResponse{}), p.err
}

var seasonOne = time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)

func TestAResetIsNamedBySeasonAndDueAtItsStart(t *testing.T) {
	asked := &planet{}

	require.NoError(t, rpc_planet_resets.New(asked).Reset(t.Context(), season.Window{Number: 1, StartsAt: seasonOne}))

	require.Len(t, asked.asked, 1)
	assert.True(t, proto.Equal(&planetv1.ResetMapRequest{ResetId: "season-1", DecidedAt: timestamppb.New(seasonOne)}, asked.asked[0]))
}

func TestAResetPlanetRefusesIsRefusedForGood(t *testing.T) {
	for code, refused := range map[connect.Code]bool{
		connect.CodeInvalidArgument: true, connect.CodeUnavailable: false, connect.CodeDeadlineExceeded: false,
	} {
		err := rpc_planet_resets.New(&planet{err: connect.NewError(code, errors.New("no"))}).
			Reset(t.Context(), season.Window{Number: 1, StartsAt: seasonOne})

		require.Error(t, err)
		assert.Equal(t, refused, errors.Is(err, moments.ErrRefused), "%s", code)
	}
}
