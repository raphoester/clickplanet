package seasonmessage

import (
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// DraftOf reads an unset duration as none: no break, and a length the calendar refuses; an unset or empty finale as
// none.
func DraftOf(brk, length *durationpb.Duration, finale *gamemasterv1.SeasonFinale) season.Draft {
	draft := season.Draft{Break: brk.AsDuration(), Length: length.AsDuration()}
	if finale != nil {
		draft.Finale = season.Finale{
			Length: finale.GetLength().AsDuration(), Set: schedule.SetName(finale.GetRuleSet()),
			Description: schedule.Description(finale.GetDescription()), Times: schedule.Times(finale.GetTimes()),
		}
	}
	return draft
}

// ScheduledOf is a season as the operator plans it, with the window the calendar works out.
func ScheduledOf(planned season.Snapshot, window season.Window) *gamemasterv1.ScheduledSeason {
	return &gamemasterv1.ScheduledSeason{
		Season: SeasonOf(window), BreakBefore: durationpb.New(planned.Break), Length: durationpb.New(planned.Length),
		Finale: FinaleOf(planned.Finale),
	}
}

// FinaleOf is a season's finale in the wire's words; nil for none.
func FinaleOf(finale season.Finale) *gamemasterv1.SeasonFinale {
	if !finale.Planned() {
		return nil
	}
	return &gamemasterv1.SeasonFinale{
		Length: durationpb.New(finale.Length), RuleSet: string(finale.Set), Description: string(finale.Description),
		Times: uint32(finale.Times),
	}
}

// ScheduledFinaleOf is a season's finale with its window.
func ScheduledFinaleOf(window season.Window) *gamemasterv1.ScheduledFinale {
	return &gamemasterv1.ScheduledFinale{
		Season: uint32(window.Number), Finale: FinaleOf(window.Finale), StartsAt: timestamppb.New(window.FinaleStartsAt()),
		EndsAt: timestamppb.New(window.EndsAt),
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
	case errors.Is(err, season.ErrStarted), errors.Is(err, season.ErrEnded), errors.Is(err, season.ErrNotLast),
		errors.Is(err, season.ErrFinaleStarted), errors.Is(err, season.ErrFinaleOverlap), errors.Is(err, schedule.ErrUnknownSet):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return err
}
