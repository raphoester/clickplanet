package seasonmessage

import (
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// DraftOf reads an unset duration as none: no break, and a length the calendar refuses.
func DraftOf(brk, length *durationpb.Duration) season.Draft {
	return season.Draft{Break: brk.AsDuration(), Length: length.AsDuration()}
}

// ScheduledOf is a season as the operator plans it, with the window the calendar works out.
func ScheduledOf(planned season.Snapshot, window season.Window) *gamemasterv1.ScheduledSeason {
	return &gamemasterv1.ScheduledSeason{
		Season: SeasonOf(window), BreakBefore: durationpb.New(planned.Break), Length: durationpb.New(planned.Length),
	}
}

func SeasonOf(window season.Window) *gamemasterv1.Season {
	return &gamemasterv1.Season{
		Number: uint32(window.Number), StartsAt: timestamppb.New(window.StartsAt), EndsAt: timestamppb.New(window.EndsAt),
		BookId: string(moments.BookOf(window.Number)),
	}
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
