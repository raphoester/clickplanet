//go:build testing

package posts

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type RecordedPause struct {
	clock  *cptime.FixedClock
	Waited []time.Duration
}

func NewRecordedPause(clock *cptime.FixedClock) *RecordedPause {
	return &RecordedPause{clock: clock}
}

var _ Pause = (*RecordedPause)(nil)

func (p *RecordedPause) Wait(_ context.Context, d time.Duration) error {
	p.Waited = append(p.Waited, d)
	p.clock.Advance(d)
	return nil
}

type ScriptedService struct {
	refusals  []error
	Refusing  map[Platform]error
	Scheduled []PostSnapshot
}

func NewScriptedService(refusals ...error) *ScriptedService {
	return &ScriptedService{refusals: refusals}
}

var _ Service = (*ScriptedService)(nil)

func (s *ScriptedService) Schedule(_ context.Context, post Post) (Scheduled, error) {
	if len(s.refusals) > 0 {
		refusal := s.refusals[0]
		s.refusals = s.refusals[1:]
		return Scheduled{}, refusal
	}
	s.Scheduled = append(s.Scheduled, post.TakeSnapshot())
	return Scheduled{ID: PostID("post-" + post.clip.title), Refused: s.Refusing}, nil
}

// A dice that always shows the same face.
type LoadedDice int

var _ Dice = LoadedDice(0)

func (d LoadedDice) IntN(n int) int {
	return int(d) % n
}

// The posts of a run as a calendar with room on every platform would place them, 2 hours apart from the lead of 5
// minutes: a test's stand-in for the publishing module.
func (p Plan) Posts(clips []Clip, now time.Time, dice Dice) []Post {
	ranked := Ranked(clips)
	planned := make([]Post, 0, len(ranked))
	at := ScriptedPlanner{}.from(now)
	for i, clip := range ranked {
		post := p.Post(clip, i+1, len(ranked), dice)
		planned = append(planned, post.At(Slot{At: at, Platforms: post.platforms}))
		at = at.Add(2 * time.Hour)
	}
	return planned
}

// A calendar that gives each post the slots it holds, in turn, then the last again; with none, the first whole minute 5
// minutes from now, on every platform asked. It records what it was asked.
type ScriptedPlanner struct {
	Slots  []Slot
	Held   bool
	Failed error
	Now    time.Time
	Asked  [][]Platform
}

var _ Planner = (*ScriptedPlanner)(nil)

func (p *ScriptedPlanner) Slot(_ context.Context, platforms []Platform) (Slot, bool, error) {
	p.Asked = append(p.Asked, platforms)
	if p.Failed != nil {
		return Slot{}, false, p.Failed
	}
	if p.Held {
		return Slot{}, false, nil
	}
	if len(p.Slots) == 0 {
		return Slot{At: p.from(p.Now), Platforms: platforms}, true, nil
	}
	slot := p.Slots[0]
	if len(p.Slots) > 1 {
		p.Slots = p.Slots[1:]
	}
	return slot, true, nil
}

func (ScriptedPlanner) from(now time.Time) time.Time {
	at := now.Add(5 * time.Minute)
	whole := at.Truncate(time.Minute)
	if whole.Before(at) {
		whole = whole.Add(time.Minute)
	}
	return whole
}
