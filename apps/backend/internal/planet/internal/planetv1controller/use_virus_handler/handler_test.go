package use_virus_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/use_virus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses/usecases/use_virus_usecase"
)

type stubUseCase struct {
	err error
	in  use_virus_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in use_virus_usecase.In) error {
	s.in = in
	return s.err
}

func use(t *testing.T, useCase use_virus_handler.UseCase) error {
	t.Helper()

	_, err := use_virus_handler.New(useCase).UseVirus(t.Context(), connect.NewRequest(&planetv1.UseVirusRequest{
		TileId: 8, CountryId: "fr",
	}))
	if err != nil {
		return fmt.Errorf("use refused: %w", err)
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

	require.NoError(t, use(t, useCase))

	assert.Equal(t, use_virus_usecase.In{TileID: 8, CountryID: "fr"}, useCase.in)
}

func TestNoEnemyTileOrAShieldedOneIsAFailedPreconditionWithAVirusRefusal(t *testing.T) {
	err := use(t, &stubUseCase{err: fmt.Errorf("tile 8: %w", clicks.ErrNotInfectable)})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.IsType(t, &planetv1.VirusRefusal{}, detailOf(t, err))
}

func TestATileOutOfReachIsAFailedPreconditionWithAReachRefusal(t *testing.T) {
	err := use(t, &stubUseCase{err: fmt.Errorf("tile 8: %w", clicks.ErrOutOfReach)})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.IsType(t, &planetv1.ReachRefusal{}, detailOf(t, err))
}

func TestUsingNoVirusIsNotFound(t *testing.T) {
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(use(t, &stubUseCase{err: use_virus_usecase.ErrNoVirus})))
}

func TestABannedCallerIsDeniedAndToldUntilWhen(t *testing.T) {
	until := time.Date(2026, 11, 16, 13, 0, 0, 0, time.UTC)
	err := use(t, &stubUseCase{err: clicks.NewRefusal(until)})

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.IsType(t, &planetv1.BanRefusal{}, detailOf(t, err))
}

func TestAMalformedUseIsAnInvalidArgument(t *testing.T) {
	for _, cause := range []error{clicks.ErrUnknownCountry, clicks.ErrTileOutOfRange} {
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(use(t, &stubUseCase{err: fmt.Errorf("%w: detail", cause)})))
	}
}
