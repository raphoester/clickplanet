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
		},
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheSetIsSavedAsSent(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, save(t, useCase))

	assert.Equal(t, []rules.Snapshot{{
		Name:  "rush",
		Tempo: 3,
		Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
	}}, useCase.asked)
}

func TestASetThatCannotPlayIsAnInvalidArgument(t *testing.T) {
	err := save(t, &stubUseCase{err: rules.ErrInvalidSet})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	require.ErrorIs(t, save(t, &stubUseCase{err: down}), down)
}
