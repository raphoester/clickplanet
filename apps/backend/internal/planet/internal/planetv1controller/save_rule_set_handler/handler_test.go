package save_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type stubUseCase struct {
	err   error
	asked []rules.Snapshot
}

func (s *stubUseCase) Execute(_ context.Context, in rules.Snapshot) error {
	s.asked = append(s.asked, in)
	return s.err
}

func save(t *testing.T, useCase *stubUseCase) error {
	t.Helper()

	_, err := save_rule_set_handler.New(useCase).SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{
		Set: &planetv1.RuleSet{
			Name:  "rush",
			Tempo: 3,
			Boxes: &planetv1.BoxPace{
				MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: 30,
			},
			Clicks:    &planetv1.ClickRules{PerSecond: 0.2, Burst: 60, NewAccountClicks: 10},
			TollSteps: []*planetv1.TollStep{{Share: 0.3, Slowdown: 4}},
			Bonuses: &planetv1.BonusRules{
				Kinds:   &planetv1.KindWeights{Bomb: 2},
				Shield:  &planetv1.ShieldRules{PerTile: 20},
				Welcome: &planetv1.WelcomeRules{Within: durationpb.New(10 * time.Minute)},
			},
			Quiz: &planetv1.QuizRules{Enabled: true, AnswerWindow: durationpb.New(8 * time.Second), LeaderBias: 1},
		},
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheSetIsSavedAsSent(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, save(t, useCase))

	require.Len(t, useCase.asked, 1)
	asked := useCase.asked[0]
	assert.Equal(t, rules.SetName("rush"), asked.Name)
	assert.InDelta(t, 3, asked.Tempo, 1e-9)
	assert.Equal(t, bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30}, asked.Bonuses.Boxes)
	assert.Equal(t, clicks.Throttle{PerSecond: 0.2, Burst: 60, NewAccountClicks: 10}, asked.Clicks)
	assert.Equal(t, []clicks.TollStep{{Share: 0.3, Slowdown: 4}}, asked.Toll)
	assert.InDelta(t, 2, asked.Bonuses.Kinds[bonuses.KindBomb], 1e-9)
	assert.Equal(t, 20, asked.Bonuses.Shield.PerTile)
	assert.Equal(t, 10*time.Minute, asked.Bonuses.Welcome.Within)
	assert.Equal(t, bonuses.QuizRules{Enabled: true, AnswerWindow: 8 * time.Second, LeaderBias: 1}, asked.Bonuses.Quiz)
}

func TestASetThatCannotPlayIsAnInvalidArgument(t *testing.T) {
	err := save(t, &stubUseCase{err: rules.ErrInvalidSet})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	require.ErrorIs(t, save(t, &stubUseCase{err: down}), down)
}
