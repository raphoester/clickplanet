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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
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
	day    = days.TestingDayAt(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
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

func charges(kind planetv1.BonusKind, amount uint32) *missionsv1.MissionReward {
	return &missionsv1.MissionReward{Kind: &missionsv1.MissionReward_Charges{Charges: &missionsv1.Charges{Kind: kind, Amount: amount}}}
}

func TestEachMissionIsAnsweredInItsKindsOwnMessage(t *testing.T) {
	brief, err := missions.BriefOf([]missions.MissionSnapshot{
		{ID: bank, Order: missions.TakeFor{Flag: "fr", Goal: 60}, Reward: missions.Reward{Kind: missions.RewardShields, Amount: 10}},
		{ID: ground, Order: missions.TakeIn{Ground: "br", Goal: 200}, Reward: missions.Reward{Kind: missions.RewardBomb, Amount: 1}},
	})
	require.NoError(t, err)
	useCase := &stubUseCase{out: get_missions_usecase.Out{
		Day: day, Brief: brief, Sheet: missions.SheetOf(missions.SheetSnapshot{
			Progress: map[missions.MissionID]uint32{bank: 60, ground: 12}, Paid: cpcolls.NewSet(bank),
		}),
	}}

	res, err := ask(t, useCase)

	require.NoError(t, err)
	assert.Equal(t, get_missions_usecase.In{Account: ada}, useCase.in)
	assert.True(t, proto.Equal(&missionsv1.GetMissionsResponse{
		EndsAtUnixMs: time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC).UnixMilli(),
		Missions: []*missionsv1.Mission{
			{
				Id: bank.String(), Done: true, Paid: true, Reward: charges(planetv1.BonusKind_BONUS_KIND_SHIELDS, 10),
				Kind: &missionsv1.Mission_TakeFor{TakeFor: &missionsv1.TakeFor{CountryId: "fr", Goal: 60, Taken: 60}},
			},
			{
				Id: ground.String(), Reward: charges(planetv1.BonusKind_BONUS_KIND_BOMB, 1),
				Kind: &missionsv1.Mission_TakeIn{TakeIn: &missionsv1.TakeIn{CountryId: "br", Goal: 200, Taken: 12}},
			},
		},
	}, res))
}

func TestEachKindOfMissionHasItsMessage(t *testing.T) {
	orders := []missions.Order{
		missions.TakeFromIn{Flag: "dz", Ground: "br", Goal: 60},
		missions.CatchBoxes{Goal: 4},
		missions.AnswerQuizzes{Goal: 3},
		missions.BombIn{Ground: "br", Goal: 2},
		missions.ShieldTiles{Shields: 10, Goal: 2},
		missions.FortifyTiles{Goal: 40},
	}
	snapshots := make([]missions.MissionSnapshot, 0, len(orders))
	progress := map[missions.MissionID]uint32{}
	for n, order := range orders {
		id := missions.MissionID{15: byte(n + 1)}
		snapshots = append(snapshots, missions.MissionSnapshot{ID: id, Order: order, Reward: missions.Reward{Kind: missions.RewardBomb, Amount: 1}})
		progress[id] = 1
	}
	brief, err := missions.BriefOf(snapshots)
	require.NoError(t, err)

	res, err := ask(t, &stubUseCase{out: get_missions_usecase.Out{
		Day: day, Brief: brief, Sheet: missions.SheetOf(missions.SheetSnapshot{Progress: progress}),
	}})

	require.NoError(t, err)
	answered := res.GetMissions()
	require.Len(t, answered, len(orders))
	kinds := []proto.Message{
		answered[0].GetTakeFromIn(), answered[1].GetCatchBoxes(), answered[2].GetAnswerQuizzes(),
		answered[3].GetBombIn(), answered[4].GetShieldTiles(), answered[5].GetFortifyTiles(),
	}
	want := []proto.Message{
		&missionsv1.TakeFromIn{CountryId: "dz", GroundId: "br", Goal: 60, Taken: 1},
		&missionsv1.CatchBoxes{Goal: 4, Caught: 1},
		&missionsv1.AnswerQuizzes{Goal: 3, Answered: 1},
		&missionsv1.BombIn{CountryId: "br", Goal: 2, Dropped: 1},
		&missionsv1.ShieldTiles{Shields: 10, Goal: 2, Shielded: 1},
		&missionsv1.FortifyTiles{Goal: 40, Fortified: 1},
	}
	for n := range want {
		assert.True(t, proto.Equal(want[n], kinds[n]), "%v", kinds[n])
	}
}

func TestAnAnswerBeforeAnyFlagHasNoMissions(t *testing.T) {
	res, err := ask(t, &stubUseCase{out: get_missions_usecase.Out{Day: day, Sheet: missions.SheetOf(missions.SheetSnapshot{})}})

	require.NoError(t, err)
	assert.Empty(t, res.GetMissions())
}

func TestAnAnswerWhenNoDayRunsHasNoEndAndNoMissions(t *testing.T) {
	res, err := ask(t, &stubUseCase{out: get_missions_usecase.Out{}})

	require.NoError(t, err)
	assert.Zero(t, res.GetEndsAtUnixMs())
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
