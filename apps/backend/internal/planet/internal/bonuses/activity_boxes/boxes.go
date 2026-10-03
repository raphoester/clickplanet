package activity_boxes

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Recorder interface {
	Record(event activity.Event)
}

func New(recorder Recorder, clock cptime.Clock) *Boxes {
	return &Boxes{recorder: recorder, clock: clock}
}

// Boxes is called with the registry locked, so it only appends and never calls back.
type Boxes struct {
	recorder Recorder
	clock    cptime.Clock
}

func (b *Boxes) Offered(scope string) { b.record(activity.KindBoxOffered, scope, 0) }

func (b *Boxes) Caught(scope string, after time.Duration) {
	b.record(activity.KindBoxCaught, scope, after)
}

func (b *Boxes) Lapsed(scope string) { b.record(activity.KindBoxLapsed, scope, 0) }

func (b *Boxes) Foreign(scope string) { b.record(activity.KindBoxForeign, scope, 0) }

func (b *Boxes) record(kind activity.Kind, scope string, delay time.Duration) {
	b.recorder.Record(activity.Event{At: b.clock.Now(), Kind: kind, Caller: activity.CallerOfScope(scope), Delay: delay})
}
