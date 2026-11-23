package lay_trap_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/lay_trap_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/traps/usecases/lay_trap_usecase"
)

type stubUseCase struct {
	err error
	in  lay_trap_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in lay_trap_usecase.In) error {
	s.in = in
	return s.err
}

func lay(t *testing.T, useCase lay_trap_handler.UseCase) error {
	t.Helper()

	_, err := lay_trap_handler.New(useCase).LayTrap(t.Context(), connect.NewRequest(&planetv1.LayTrapRequest{
		TileId: 8, CountryId: "fr",
	}))
	if err != nil {
		return fmt.Errorf("lay refused: %w", err)
	}
	return nil
}

func detailOf(t *testing.T, err error) proto.Message {
	t.Helper()
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	detail, detailErr := refused.Details()[0].Value()
	require.NoError(t, detailErr)
	return detail
}

func TestTheTileAndTheFlagReachTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, lay(t, useCase))

	assert.Equal(t, lay_trap_usecase.In{TileID: 8, CountryID: "fr"}, useCase.in)
}

func TestATileNotTheFlagsOrTrappedIsAFailedPreconditionWithATrapRefusal(t *testing.T) {
	err := lay(t, &stubUseCase{err: fmt.Errorf("tile 8: %w", clicks.ErrNotTrappable)})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.IsType(t, &planetv1.TrapRefusal{}, detailOf(t, err))
}

func TestLayingNoTrapIsNotFound(t *testing.T) {
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(lay(t, &stubUseCase{err: lay_trap_usecase.ErrNoTrap})))
}

func TestABannedCallerIsDeniedAndToldUntilWhen(t *testing.T) {
	until := time.Date(2026, 11, 16, 13, 0, 0, 0, time.UTC)
	err := lay(t, &stubUseCase{err: clicks.NewRefusal(until)})

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.IsType(t, &planetv1.BanRefusal{}, detailOf(t, err))
}

func TestAMalformedLayIsAnInvalidArgument(t *testing.T) {
	for _, cause := range []error{clicks.ErrUnknownCountry, clicks.ErrTileOutOfRange} {
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(lay(t, &stubUseCase{err: fmt.Errorf("%w: detail", cause)})))
	}
}
