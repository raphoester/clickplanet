package catalog

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
)

var (
	ErrUnknownFlag = errors.New("no flag was opened with this id")
	ErrClosed      = errors.New("the flag is closed already")
)

type State string

const (
	Open   State = "open"
	Closed State = "closed"
)

// Flag is a flag opened since the game shipped. It is closed, never deleted: history still names it.
type Flag struct {
	id        FlagID
	name      Name
	state     State
	decidedAt time.Time
}

func (f Flag) Closed(at time.Time) (Flag, error) {
	if f.state == Closed {
		return f, fmt.Errorf("%s: %w", f.id, ErrClosed)
	}
	f.state, f.decidedAt = Closed, at
	return f, nil
}

// Facts is what a module that never heard of the flag must hear to know it as it is now.
func (f Flag) Facts() []proto.Message {
	switch f.state {
	case Open:
		return []proto.Message{&flagsv1.FlagOpened{
			Flag: f.id.String(), Name: f.name.String(), OpenedAt: timestamppb.New(f.decidedAt),
		}}
	case Closed:
		return []proto.Message{&flagsv1.FlagClosed{Flag: f.id.String(), ClosedAt: timestamppb.New(f.decidedAt)}}
	}
	return nil
}

func (f Flag) TakeSnapshot() Snapshot {
	return Snapshot{ID: f.id, Name: f.name, State: f.state, DecidedAt: f.decidedAt}
}

type Snapshot struct {
	ID   FlagID
	Name Name
	// Since DecidedAt.
	State     State
	DecidedAt time.Time
}

func (s Snapshot) Restore() (Flag, error) {
	switch s.State {
	case Open, Closed:
		return Flag{id: s.ID, name: s.Name, state: s.State, decidedAt: s.DecidedAt}, nil
	}
	return Flag{}, fmt.Errorf("the flag %s is %q, which is no state", s.ID, s.State)
}
