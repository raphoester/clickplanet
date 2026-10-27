package seasonmessage

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// DraftOf reads an unset time as none, which the calendar refuses.
func DraftOf(startsAt, endsAt *timestamppb.Timestamp) season.Draft {
	return season.Draft{StartsAt: timeOf(startsAt), EndsAt: timeOf(endsAt)}
}

func timeOf(at *timestamppb.Timestamp) time.Time {
	if at == nil {
		return time.Time{}
	}
	return at.AsTime()
}

// Refusal is the code of what the calendar refused, and the error as it is for anything else.
func Refusal(err error) error {
	switch {
	case errors.Is(err, season.ErrInvalidSeason):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, season.ErrUnknownSeason):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, season.ErrStarted), errors.Is(err, season.ErrEnded), errors.Is(err, season.ErrNotLast):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return err
}
