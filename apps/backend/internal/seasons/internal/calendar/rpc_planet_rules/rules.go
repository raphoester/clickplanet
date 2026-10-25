package rpc_planet_rules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/play_phase_usecase"
)

type Planet interface {
	ApplyRuleSet(
		ctx context.Context,
		req *connect.Request[planetv1.ApplyRuleSetRequest],
	) (*connect.Response[planetv1.ApplyRuleSetResponse], error)
}

// Config names, in planet's words, the rule set each stage of a season plays by. Unset is planet's "default".
type Config struct {
	Days   string
	Finale string
	Over   string
}

const planetDefault = "default"

var ErrUnknownStage = errors.New("a season has no such stage")

const askTimeout = 2 * time.Second

func New(planet Planet, config Config) *Rules {
	return &Rules{planet: planet, config: config}
}

type Rules struct {
	planet Planet
	config Config
}

var _ play_phase_usecase.Game = (*Rules)(nil)

func (r *Rules) Play(ctx context.Context, phase calendar.Phase) error {
	set, err := r.config.setOf(phase.Stage)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	// The phase is the request: sent again after a boot, planet keeps the change it made the first time.
	request := fmt.Sprintf("season-%d:%s", phase.Season, phase.Stage)
	if _, err := r.planet.ApplyRuleSet(ctx, connect.NewRequest(&planetv1.ApplyRuleSetRequest{
		RequestId: request, RuleSet: set,
	})); err != nil {
		return fmt.Errorf("failed to ask the planet module for rule set %q, for %s: %w", set, request, err)
	}

	return nil
}

func (c Config) setOf(stage calendar.Stage) (string, error) {
	var set string
	switch stage {
	case calendar.StageDays:
		set = c.Days
	case calendar.StageFinale:
		set = c.Finale
	case calendar.StageOver:
		set = c.Over
	default:
		return "", fmt.Errorf("%q: %w", stage, ErrUnknownStage)
	}

	if set == "" {
		return planetDefault, nil
	}
	return set, nil
}
