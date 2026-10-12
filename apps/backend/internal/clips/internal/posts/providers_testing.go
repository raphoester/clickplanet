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
	Scheduled []PostSnapshot
}

func NewScriptedService(refusals ...error) *ScriptedService {
	return &ScriptedService{refusals: refusals}
}

var _ Service = (*ScriptedService)(nil)

func (s *ScriptedService) Schedule(_ context.Context, post Post) (PostID, error) {
	if len(s.refusals) > 0 {
		refusal := s.refusals[0]
		s.refusals = s.refusals[1:]
		return "", refusal
	}
	s.Scheduled = append(s.Scheduled, post.TakeSnapshot())
	return PostID("post-" + post.clip.title), nil
}
