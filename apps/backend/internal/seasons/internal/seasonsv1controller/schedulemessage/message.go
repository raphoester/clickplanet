package schedulemessage

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
)

// DraftOf reads an unset time as none, which the schedule refuses.
func DraftOf(set string, startsAt, endsAt *timestamppb.Timestamp, priority uint32) schedule.Draft {
	return schedule.Draft{
		Set: schedule.SetName(set), StartsAt: timeOf(startsAt), EndsAt: timeOf(endsAt), Priority: schedule.Priority(priority),
	}
}

func timeOf(at *timestamppb.Timestamp) time.Time {
	if at == nil {
		return time.Time{}
	}
	return at.AsTime()
}

// Refusal is the code of what the schedule refused, and the error as it is for anything else.
func Refusal(err error) error {
	switch {
	case errors.Is(err, schedule.ErrInvalidEntry):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, schedule.ErrUnknownEntry):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, schedule.ErrStarted), errors.Is(err, schedule.ErrUnknownSet):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return err
}
