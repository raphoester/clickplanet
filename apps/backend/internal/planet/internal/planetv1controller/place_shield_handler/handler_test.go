package place_shield_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/place_shield_handler"
)

type stubUseCase struct {
	held bonuses.Held
	err  error
	in   place_shield_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in place_shield_usecase.In) (bonuses.Held, error) {
	s.in = in
	return s.held, s.err
}

func place(t *testing.T, useCase *stubUseCase) (*planetv1.PlaceShieldResponse, error) {
	t.Helper()

	res, err := place_shield_handler.New(useCase).PlaceShield(t.Context(),
		connect.NewRequest(&planetv1.PlaceShieldRequest{TileId: 7, CountryId: "fr"}))
	if err != nil {
		return nil, fmt.Errorf("placing refused: %w", err)
	}

	return res.Msg, nil
}

func TestAShieldPlacedAnswersWhatIsLeftInHand(t *testing.T) {
	useCase := &stubUseCase{held: bonuses.Held{Shields: 4}}

	msg, err := place(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"}, useCase.in)
	assert.Equal(t, uint32(4), msg.GetCharges().GetShields())
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		clicks.ErrUnknownCountry:         connect.CodeInvalidArgument,
		clicks.ErrTileOutOfRange:         connect.CodeInvalidArgument,
		clicks.ErrNotYourTile:            connect.CodeFailedPrecondition,
		clicks.ErrTileFull:               connect.CodeFailedPrecondition,
		place_shield_usecase.ErrNoShield: connect.CodeNotFound,
	} {
		_, refused := place(t, &stubUseCase{err: err})
		assert.Equal(t, code, connect.CodeOf(refused), err.Error())
	}
}
