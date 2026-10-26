package schedule

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

var ErrUnknownStage = errors.New("a season has no such stage")

// PhaseSets names the set each stage of a season plays, at priority 0: the config's seasons.ruleSets.
type PhaseSets struct {
	Days   SetName
	Finale SetName
	Over   SetName
}

func (p PhaseSets) Validate() error {
	var errs []error
	for _, stage := range []calendar.Stage{calendar.StageDays, calendar.StageFinale, calendar.StageOver} {
		set, err := p.SetOf(stage)
		if err == nil {
			err = set.check()
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", stage, err))
		}
	}
	return errors.Join(errs...)
}

func (p PhaseSets) SetOf(stage calendar.Stage) (SetName, error) {
	switch stage {
	case calendar.StageDays:
		return p.Days, nil
	case calendar.StageFinale:
		return p.Finale, nil
	case calendar.StageOver:
		return p.Over, nil
	}
	return "", fmt.Errorf("%q: %w", stage, ErrUnknownStage)
}
