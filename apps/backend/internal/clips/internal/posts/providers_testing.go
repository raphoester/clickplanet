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

// The posts of a run on an empty feed, each planned once the one before it is taken everywhere, as the run plans them.
func (p Plan) Posts(clips []Clip, now time.Time, dice Dice) []Post {
	var (
		planned []Post
		feed    Feed
	)
	ranked := Ranked(clips)
	for i, clip := range ranked {
		post, room := p.Next(clip, i+1, len(ranked), feed, now, dice)
		if !room {
			continue
		}
		planned = append(planned, post)
		feed = feed.With(post, Scheduled{ID: "planned"})
	}
	return planned
}
