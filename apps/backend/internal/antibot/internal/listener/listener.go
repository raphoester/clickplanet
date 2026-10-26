package listener

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Guard interface {
	Attempted(click guard.Click)
	Inspect(ctx context.Context, click guard.Click)
	Committed(click guard.Click)
	Caught(scope string, after time.Duration)
	Missed(scope string)
	Foreign(scope string)
	Fetched(scope string, maps float64, offMap bool)
	Listened(scope string)
}

var ErrUnknownOutcome = errors.New("a click tried with an outcome the antibot does not know")

func New(guard Guard, marks *Marks) *Listener {
	return &Listener{guard: guard, marks: marks}
}

type Listener struct {
	guard Guard
	marks *Marks
}

func (l *Listener) ClickTried() cpredis.Handler[*planetv1.ClickTried] {
	return heard[*planetv1.ClickTried]{listener: l, at: (*planetv1.ClickTried).GetTriedAt, tell: l.tried}
}

func (l *Listener) MapRead() cpredis.Handler[*planetv1.MapRead] {
	return heard[*planetv1.MapRead]{listener: l, at: (*planetv1.MapRead).GetReadAt, tell: l.read}
}

func (l *Listener) StreamOpened() cpredis.Handler[*planetv1.StreamOpened] {
	return heard[*planetv1.StreamOpened]{listener: l, at: (*planetv1.StreamOpened).GetOpenedAt, tell: l.opened}
}

func (l *Listener) BoxCaught() cpredis.Handler[*planetv1.BoxCaught] {
	return heard[*planetv1.BoxCaught]{listener: l, at: (*planetv1.BoxCaught).GetCaughtAt, tell: l.caught}
}

func (l *Listener) BoxLapsed() cpredis.Handler[*planetv1.BoxLapsed] {
	return heard[*planetv1.BoxLapsed]{listener: l, at: (*planetv1.BoxLapsed).GetLapsedAt, tell: l.lapsed}
}

func (l *Listener) ForeignBoxClaimed() cpredis.Handler[*planetv1.ForeignBoxClaimed] {
	return heard[*planetv1.ForeignBoxClaimed]{listener: l, at: (*planetv1.ForeignBoxClaimed).GetClaimedAt, tell: l.foreign}
}

func (l *Listener) tried(ctx context.Context, fact *planetv1.ClickTried) error {
	click := clickOf(fact)

	switch outcome := fact.GetOutcome(); outcome {
	case planetv1.ClickOutcome_CLICK_OUTCOME_REFUSED:
	case planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, planetv1.ClickOutcome_CLICK_OUTCOME_OTHER_FLAG:
		l.guard.Attempted(click)
	case planetv1.ClickOutcome_CLICK_OUTCOME_INVALID, planetv1.ClickOutcome_CLICK_OUTCOME_FAILED:
		l.guard.Attempted(click)
		l.guard.Inspect(ctx, click)
	case planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN, planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK,
		planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED:
		l.guard.Attempted(click)
		l.guard.Inspect(ctx, click)
		l.guard.Committed(click)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownOutcome, outcome)
	}
	return nil
}

func clickOf(fact *planetv1.ClickTried) guard.Click {
	click := guard.Click{
		Scope:    fact.GetScope(),
		Account:  fact.GetAccountId(),
		SignedIn: fact.GetLinked(),
		Tile:     fact.GetTileId(),
		Country:  fact.GetCountry(),
		At:       fact.GetTriedAt().AsTime(),
		Held:     fact.GetOwner(),
		NoOp:     fact.GetOutcome() == planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED,
		Shielded: fact.GetOutcome() == planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK,
	}
	if at := fact.GetPosition(); at != nil {
		click.Position = guard.Point{X: at.GetX(), Y: at.GetY(), Z: at.GetZ()}
	}
	return click
}

func (l *Listener) read(_ context.Context, fact *planetv1.MapRead) error {
	l.guard.Fetched(fact.GetScope(), fact.GetMaps(), fact.GetOffMap())
	return nil
}

func (l *Listener) opened(_ context.Context, fact *planetv1.StreamOpened) error {
	l.guard.Listened(fact.GetScope())
	return nil
}

func (l *Listener) caught(_ context.Context, fact *planetv1.BoxCaught) error {
	l.guard.Caught(fact.GetScope(), fact.GetAfter().AsDuration())
	return nil
}

func (l *Listener) lapsed(_ context.Context, fact *planetv1.BoxLapsed) error {
	l.guard.Missed(fact.GetScope())
	return nil
}

func (l *Listener) foreign(_ context.Context, fact *planetv1.ForeignBoxClaimed) error {
	l.guard.Foreign(fact.GetScope())
	return nil
}

type heard[T proto.Message] struct {
	listener *Listener
	at       func(T) *timestamppb.Timestamp
	tell     func(context.Context, T) error
}

func (h heard[T]) Handle(ctx context.Context, fact T) error {
	return h.listener.marks.once(string(fact.ProtoReflect().Descriptor().FullName()), h.at(fact).AsTime(), func() error {
		return h.tell(ctx, fact)
	})
}
