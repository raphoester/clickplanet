package get_map_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map"
)

type stubUseCase struct {
	batch clicks.DenseBatch
	err   error
	seen  []get_map.In
}

func (s *stubUseCase) Execute(_ context.Context, in get_map.In) (clicks.DenseBatch, error) {
	s.seen = append(s.seen, in)
	return s.batch, s.err
}

func getMap(
	t *testing.T,
	useCase *stubUseCase,
	req *planetv1.GetMapRequest,
) (*connect.Response[planetv1.GetMapResponse], error) {
	t.Helper()
	return get_map_handler.New(useCase).GetMap(t.Context(), connect.NewRequest(req))
}

func TestGetMapMapsTheRequest(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := getMap(t, useCase, &planetv1.GetMapRequest{StartTileId: 7, EndTileId: 9})

	require.NoError(t, err)
	require.Equal(t, []get_map.In{{Start: 7, End: 9}}, useCase.seen)
}

func TestGetMapMapsTheBatch(t *testing.T) {
	useCase := &stubUseCase{batch: clicks.DenseBatch{
		Start: 7,
		Codes: []string{"", "fr"},
		Tiles: []byte{0x01, 0x00, 0x00, 0x00},
	}}

	res, err := getMap(t, useCase, &planetv1.GetMapRequest{})

	require.NoError(t, err)
	assert.Equal(t, uint32(7), res.Msg.GetStartTileId())
	assert.Equal(t, []string{"", "fr"}, res.Msg.GetCodes())
	assert.Equal(t, []byte{0x01, 0x00, 0x00, 0x00}, res.Msg.GetTiles())
}

func TestGetMapMapsTheErrors(t *testing.T) {
	t.Run("a span the map cannot answer is the caller's fault", func(t *testing.T) {
		_, err := getMap(t, &stubUseCase{err: clicks.ErrInvalidTileRange}, &planetv1.GetMapRequest{})

		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorIs(t, err, clicks.ErrInvalidTileRange)
	})

	t.Run("anything else is left for the error interceptor", func(t *testing.T) {
		cause := errors.New("disk on fire")

		_, err := getMap(t, &stubUseCase{err: cause}, &planetv1.GetMapRequest{})

		require.ErrorIs(t, err, cause)
		require.Equal(t, connect.CodeUnknown, connect.CodeOf(err),
			"the handler picked no code: dressing this up would hide it from the interceptor's log")
	})
}
