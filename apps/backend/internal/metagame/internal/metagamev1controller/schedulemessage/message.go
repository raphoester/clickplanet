package schedulemessage

import (
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
)

// DraftOf reads an unset time as none, which the schedule refuses; a kind the wire does not name is refused here.
func DraftOf(
	set string, kind metagamev1.EntryKind, startsAt, endsAt *timestamppb.Timestamp, priority uint32,
) (schedule.Draft, error) {
	known, err := kindOf(kind)
	if err != nil {
		return schedule.Draft{}, err
	}
	return schedule.Draft{
		Set: schedule.SetName(set), Kind: known, StartsAt: timeOf(startsAt), EndsAt: timeOf(endsAt),
		Priority: schedule.Priority(priority),
	}, nil
}

func kindOf(kind metagamev1.EntryKind) (schedule.Kind, error) {
	switch kind {
	case metagamev1.EntryKind_ENTRY_KIND_BATTLE:
		return schedule.Battle, nil
	case metagamev1.EntryKind_ENTRY_KIND_PLAIN:
		return schedule.Plain, nil
	case metagamev1.EntryKind_ENTRY_KIND_UNSPECIFIED:
	}
	return "", fmt.Errorf("%w: %q", schedule.ErrUnknownKind, kind.String())
}

// KindOf is the wire's word for a kept kind, and an error for one this build does not know.
func KindOf(kind schedule.Kind) (metagamev1.EntryKind, error) {
	switch kind {
	case schedule.Battle:
		return metagamev1.EntryKind_ENTRY_KIND_BATTLE, nil
	case schedule.Plain:
		return metagamev1.EntryKind_ENTRY_KIND_PLAIN, nil
	}
	return metagamev1.EntryKind_ENTRY_KIND_UNSPECIFIED, fmt.Errorf("%w: %q", schedule.ErrUnknownKind, kind)
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
	case errors.Is(err, schedule.ErrInvalidEntry), errors.Is(err, schedule.ErrUnknownKind):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, schedule.ErrUnknownEntry):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, schedule.ErrStarted), errors.Is(err, schedule.ErrUnknownSet):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return err
}
