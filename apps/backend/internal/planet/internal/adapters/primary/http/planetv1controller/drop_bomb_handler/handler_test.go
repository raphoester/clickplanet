package drop_bomb_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb"
)

type stubUseCase struct {
	err error
	in  drop_bomb.In
}

func (s *stubUseCase) Execute(_ context.Context, in drop_bomb.In) (clicks.Blast, error) {
	s.in = in
	return clicks.Blast{}, s.err
}

func drop(t *testing.T, useCase drop_bomb_handler.UseCase) error {
	t.Helper()

	_, err := drop_bomb_handler.New(useCase).DropBomb(t.Context(), connect.NewRequest(&planetv1.DropBombRequest{
		Target:    &planetv1.GlobePoint{X: 1, Y: 2, Z: 3},
		CountryId: "fr",
	}))
	if err != nil {
		return fmt.Errorf("drop refused: %w", err)
	}

	return nil
}

func TestTheTargetAndCountryReachTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, drop(t, useCase))

	assert.Equal(t, clicks.Vec3{X: 1, Y: 2, Z: 3}, useCase.in.Target)
	assert.Equal(t, "fr", useCase.in.CountryID)
}

func TestDroppingWithNoBombIsNotFound(t *testing.T) {
	err := drop(t, &stubUseCase{err: drop_bomb.ErrNoBomb})

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAMalformedDropIsAnInvalidArgument(t *testing.T) {
	for _, cause := range []error{clicks.ErrUnknownCountry, clicks.ErrTileOutOfRange} {
		err := drop(t, &stubUseCase{err: fmt.Errorf("%w: detail", cause)})

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
}

func TestWithBoxesOffDroppingIsUnimplemented(t *testing.T) {
	err := drop(t, nil)

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}
