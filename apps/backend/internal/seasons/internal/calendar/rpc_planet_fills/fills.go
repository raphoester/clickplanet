package rpc_planet_fills

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
	FillCharges(
		ctx context.Context,
		req *connect.Request[planetv1.FillChargesRequest],
	) (*connect.Response[planetv1.FillChargesResponse], error)
}

// Config names the stages of a season that fill every player's charges when they start.
type Config struct {
	Days   bool
	Finale bool
	Over   bool
}

var ErrUnknownStage = errors.New("a season has no such stage")

const askTimeout = 2 * time.Second

func New(planet Planet, config Config) *Fills {
	return &Fills{planet: planet, config: config}
}

type Fills struct {
	planet Planet
	config Config
}

var _ play_phase_usecase.Game = (*Fills)(nil)

func (f *Fills) Play(ctx context.Context, phase calendar.Phase) error {
	fills, err := f.config.fills(phase.Stage)
	if err != nil || !fills {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	// The phase is the request: sent again after a boot, or while the fill runs, planet gives each hand once.
	request := fmt.Sprintf("season-%d:%s:fill", phase.Season, phase.Stage)
	if _, err := f.planet.FillCharges(ctx, connect.NewRequest(&planetv1.FillChargesRequest{
		RequestId: request,
	})); err != nil {
		return fmt.Errorf("failed to ask the planet module to fill every hand, for %s: %w", request, err)
	}

	return nil
}

func (c Config) fills(stage calendar.Stage) (bool, error) {
	switch stage {
	case calendar.StageDays:
		return c.Days, nil
	case calendar.StageFinale:
		return c.Finale, nil
	case calendar.StageOver:
		return c.Over, nil
	default:
		return false, fmt.Errorf("%q: %w", stage, ErrUnknownStage)
	}
}
