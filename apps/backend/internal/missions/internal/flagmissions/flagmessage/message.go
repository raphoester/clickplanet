package flagmessage

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
)

var ErrUnknownKind = errors.New("no message for this kind of flag mission")

// A flag's mission as its players see it: the browser reads it, and the other modules hear it done.
func Mission(status flagmissions.Status) (*missionsv1.FlagMission, error) {
	encoded := &missionsv1.FlagMission{Id: status.Mission.ID.String(), Points: status.Mission.Points, Done: status.Done}

	switch order := status.Mission.Order.(type) {
	case flagmissions.TakeIn:
		encoded.Kind = &missionsv1.FlagMission_TakeIn{TakeIn: &missionsv1.TakeIn{
			CountryId: order.Ground, Goal: order.Goal, Taken: status.Progress,
		}}
	case flagmissions.BreakShields:
		encoded.Kind = &missionsv1.FlagMission_BreakShields{BreakShields: &missionsv1.BreakShields{
			CountryId: order.Flag, Goal: order.Goal, Broken: status.Progress,
		}}
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownKind, order)
	}
	return encoded, nil
}

// A mission is done once, so each is told once: at the time of the fact that finished it.
func Done(flag string, done []flagmissions.Done) ([]proto.Message, error) {
	told := make([]proto.Message, 0, len(done))
	for _, each := range done {
		mission, err := Mission(each.Status())
		if err != nil {
			return nil, err
		}
		told = append(told, &missionsv1.FlagMissionDone{Flag: flag, Mission: mission, DoneAt: timestamppb.New(each.At)})
	}
	return told, nil
}
