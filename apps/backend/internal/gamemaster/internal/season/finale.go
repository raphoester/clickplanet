package season

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

var ErrFinaleStarted = errors.New("this season's finale has started: its end and its finale stay")

// Finale is a season's last battle: it ends at the season's end and starts its length before. A season with none has
// the zero Finale.
type Finale struct {
	Length      time.Duration
	Set         schedule.SetName
	Description schedule.Description
	Times       schedule.Times
}

func (f Finale) Planned() bool { return f != Finale{} }

func (f Finale) check(season time.Duration) error {
	if !f.Planned() {
		return nil
	}
	var found []error
	if f.Length <= 0 {
		found = append(found, fmt.Errorf("its finale lasts %s, not above none", f.Length))
	}
	if f.Length > season {
		found = append(found, fmt.Errorf("its finale lasts %s, longer than the season's %s", f.Length, season))
	}
	if err := f.Set.Check(); err != nil {
		found = append(found, err)
	}
	if err := f.Description.Check(); err != nil {
		found = append(found, err)
	}
	if err := f.Times.Check(schedule.Battle); err != nil {
		found = append(found, err)
	}
	if len(found) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidSeason, errors.Join(found...))
	}
	return nil
}

// FinaleStartsAt is when the season's finale starts: its length before the season's end.
func (w Window) FinaleStartsAt() time.Time {
	return w.EndsAt.Add(-w.Finale.Length)
}

func (w Window) finaleStarted(at time.Time) bool {
	return w.Finale.Planned() && !at.Before(w.FinaleStartsAt())
}

// Finales are the seasons' finales, as the schedule plays them.
func (c Calendar) Finales() []schedule.Finale {
	var finales []schedule.Finale
	for _, window := range c.Windows() {
		if !window.Finale.Planned() {
			continue
		}
		finales = append(finales, schedule.Finale{
			Set: window.Finale.Set, Description: window.Finale.Description, StartsAt: window.FinaleStartsAt(),
			EndsAt: window.EndsAt,
		})
	}
	return finales
}
