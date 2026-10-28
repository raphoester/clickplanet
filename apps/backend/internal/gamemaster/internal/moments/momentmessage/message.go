package momentmessage

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

// DebriefOf is a moment in the players' words.
func DebriefOf(moment moments.Moment) (*gamemasterv1.Debrief, error) {
	switch told := moment.(type) {
	case moments.DayEnded:
		return &gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Day{Day: dayEnded(told)}}, nil
	case moments.BattleEnded:
		return &gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Battle{Battle: battleEnded(told)}}, nil
	case moments.SeasonEnded:
		return &gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Season{Season: seasonEnded(told)}}, nil
	}
	return nil, fmt.Errorf("no debrief tells %T", moment)
}

// EventOf is a moment as the gamemaster.v1 event the other modules hear.
func EventOf(moment moments.Moment) (proto.Message, error) {
	switch told := moment.(type) {
	case moments.DayEnded:
		return dayEnded(told), nil
	case moments.BattleEnded:
		return battleEnded(told), nil
	case moments.SeasonEnded:
		return seasonEnded(told), nil
	}
	return nil, fmt.Errorf("no event tells %T", moment)
}

func dayEnded(told moments.DayEnded) *gamemasterv1.DayEnded {
	return &gamemasterv1.DayEnded{
		Season: uint32(told.Day.Season), Day: uint32(told.Day.Number), RoundId: string(moments.DayRoundOf(told.Day)),
		EndedAt: timestamppb.New(told.EndedAt),
	}
}

func battleEnded(told moments.BattleEnded) *gamemasterv1.BattleEnded {
	return &gamemasterv1.BattleEnded{
		Season: uint32(told.Battle.Season), Number: told.Battle.Number, EntryId: told.Battle.Entry.String(),
		Times: uint32(told.Battle.Times), EndsSeason: told.Battle.EndsSeason, RoundId: string(told.Battle.Round()),
		EndedAt: timestamppb.New(told.EndedAt),
	}
}

func seasonEnded(told moments.SeasonEnded) *gamemasterv1.SeasonEnded {
	ended := &gamemasterv1.SeasonEnded{
		Season: uint32(told.Season), BookId: string(told.Book), EndedAt: timestamppb.New(told.EndedAt),
		FinaleRoundId: string(told.Finale),
	}
	if told.FinaleBattle != nil {
		ended.FinaleBattle = battleEnded(*told.FinaleBattle)
	}
	return ended
}
