package collect_mission_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/usecases/collect_mission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/collect_mission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	in  collect_mission_usecase.In
	out collect_mission_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in collect_mission_usecase.In) (collect_mission_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

var ada = players.AccountID{15: 1}

func collect(ctx context.Context, useCase *stubUseCase, slot uint32) (*playerv1.CollectMissionResponse, error) {
	res, err := collect_mission_handler.New(useCase).CollectMission(ctx,
		connect.NewRequest(&playerv1.CollectMissionRequest{Slot: slot}))
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect error.
	}
	return res.Msg, nil
}

func TestTheRewardKeptIsAnswered(t *testing.T) {
	useCase := &stubUseCase{out: collect_mission_usecase.Out{Reward: missions.Reward{Kind: missions.RewardSpreadClicks, Amount: 3}}}
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(t.Context(), ada.String()))

	res, err := collect(ctx, useCase, 2)

	require.NoError(t, err)
	assert.Equal(t, collect_mission_usecase.In{Account: ada, Linked: true, Slot: 2}, useCase.in)
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, res.GetKind())
	assert.Equal(t, uint32(3), res.GetAmount())
}

func TestAGuestsTokenSaysItIsNotLinked(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := collect(cpctx.AddAccountToContext(t.Context(), ada.String()), useCase, 1)

	require.NoError(t, err)
	assert.False(t, useCase.in.Linked)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	ctx := cpctx.AddAccountToContext(t.Context(), ada.String())
	for err, code := range map[error]connect.Code{
		collect_mission_usecase.ErrNotLinked:                   connect.CodePermissionDenied,
		fmt.Errorf("%w: 4", missions.ErrNoSlot):                connect.CodeInvalidArgument,
		missions.ErrNotDone:                                    connect.CodeFailedPrecondition,
		missions.ErrCollected:                                  connect.CodeFailedPrecondition,
		fmt.Errorf("x: %w", collect_mission_usecase.ErrNoRoom): connect.CodeFailedPrecondition,
		errors.New("boom"):                                     connect.CodeUnknown,
	} {
		_, got := collect(ctx, &stubUseCase{err: err}, 1)

		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	_, err := collect(t.Context(), &stubUseCase{}, 1)

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
