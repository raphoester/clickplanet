package schedulemessage

import (
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// DraftOf reads an unset time as none, which the schedule refuses; a kind the wire does not name is refused here.
func DraftOf(
	set string, kind gamemasterv1.EntryKind, description string, startsAt, endsAt *timestamppb.Timestamp, priority, times uint32,
) (schedule.Draft, error) {
	known, err := kindOf(kind)
	if err != nil {
		return schedule.Draft{}, err
	}
	return schedule.Draft{
		Set: schedule.SetName(set), Kind: known, Description: schedule.Description(description),
		StartsAt: timeOf(startsAt), EndsAt: timeOf(endsAt), Priority: schedule.Priority(priority), Times: schedule.Times(times),
	}, nil
}

func kindOf(kind gamemasterv1.EntryKind) (schedule.Kind, error) {
	switch kind {
	case gamemasterv1.EntryKind_ENTRY_KIND_PLAIN:
		return schedule.Plain, nil
	case gamemasterv1.EntryKind_ENTRY_KIND_BATTLE:
		return schedule.Battle, nil
	case gamemasterv1.EntryKind_ENTRY_KIND_UNSPECIFIED:
	}
	return "", fmt.Errorf("%w: %q", schedule.ErrUnknownKind, kind.String())
}

// KindOf is the wire's word for a kept kind, and an error for one this build does not know.
func KindOf(kind schedule.Kind) (gamemasterv1.EntryKind, error) {
	switch kind {
	case schedule.Plain:
		return gamemasterv1.EntryKind_ENTRY_KIND_PLAIN, nil
	case schedule.Battle:
		return gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, nil
	}
	return gamemasterv1.EntryKind_ENTRY_KIND_UNSPECIFIED, fmt.Errorf("%w: %q", schedule.ErrUnknownKind, kind)
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
	case errors.Is(err, schedule.ErrInvalidEntry), errors.Is(err, schedule.ErrUnknownKind), errors.Is(err, schedule.ErrLongDescription):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, schedule.ErrUnknownEntry):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, schedule.ErrStarted), errors.Is(err, schedule.ErrUnknownSet), errors.Is(err, season.ErrFinaleOverlap):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return err
}
