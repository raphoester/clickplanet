package get_missions_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_missions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	in  get_missions_usecase.In
	out get_missions_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in get_missions_usecase.In) (get_missions_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

var (
	ada    = missions.AccountID{15: 1}
	bank   = missions.MissionID{15: 1}
	ground = missions.MissionID{15: 3}
	day    = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour}).DayOf(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
)

func ask(t *testing.T, useCase *stubUseCase) (*missionsv1.GetMissionsResponse, error) {
	t.Helper()

	res, err := get_missions_handler.New(useCase).GetMissions(cpctx.AddAccountToContext(t.Context(), ada.String()),
		connect.NewRequest(&missionsv1.GetMissionsRequest{}))
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect error.
	}
	return res.Msg, nil
}

func TestEachMissionIsAnsweredWithHowFarItWent(t *testing.T) {
	brief, err := missions.BriefOf([]missions.MissionSnapshot{
		{ID: bank, Kind: missions.KindTakeFor, Target: "fr", Goal: 60, Reward: missions.Reward{Kind: missions.RewardShields, Amount: 10}},
		{ID: ground, Kind: missions.KindTakeIn, Target: "br", Goal: 200, Reward: missions.Reward{Kind: missions.RewardBomb, Amount: 1}},
	})
	require.NoError(t, err)
	useCase := &stubUseCase{out: get_missions_usecase.Out{
		Day: day, Flag: "fr", Brief: brief, Sheet: missions.SheetOf(missions.SheetSnapshot{
			Flag: "fr", Taken: map[missions.MissionID]uint32{bank: 60, ground: 12}, Paid: cpcolls.NewSet(bank),
		}),
	}}

	res, err := ask(t, useCase)

	require.NoError(t, err)
	assert.Equal(t, get_missions_usecase.In{Account: ada}, useCase.in)
	assert.True(t, proto.Equal(&missionsv1.GetMissionsResponse{
		CountryId: "fr", Enlisted: true, EndsAtUnixMs: time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC).UnixMilli(),
		Missions: []*missionsv1.Mission{
			{
				Id: bank.String(), Kind: missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, TargetCountryId: "fr", Goal: 60, Progress: 60,
				Paid: true, RewardKind: planetv1.BonusKind_BONUS_KIND_SHIELDS, RewardAmount: 10,
			},
			{
				Id: ground.String(), Kind: missionsv1.MissionKind_MISSION_KIND_TAKE_IN, TargetCountryId: "br", Goal: 200, Progress: 12,
				RewardKind: planetv1.BonusKind_BONUS_KIND_BOMB, RewardAmount: 1,
			},
		},
	}, res))
}

func TestAnAnswerBeforeAnyFlagHasNoMissions(t *testing.T) {
	res, err := ask(t, &stubUseCase{out: get_missions_usecase.Out{Day: day, Sheet: missions.SheetOf(missions.SheetSnapshot{})}})

	require.NoError(t, err)
	assert.False(t, res.GetEnlisted())
	assert.Empty(t, res.GetMissions())
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("planet is down")

	_, err := ask(t, &stubUseCase{err: failure})

	assert.ErrorIs(t, err, failure)
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	_, err := get_missions_handler.New(&stubUseCase{}).GetMissions(t.Context(),
		connect.NewRequest(&missionsv1.GetMissionsRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
