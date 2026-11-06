package flag_mission_done_subscriber

import (
	"context"
	"errors"
	"fmt"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*missionsv1.FlagMissionDone] = Subscriber{}

var (
	errNoTime = errors.New("the flag mission has no time")
	errNoFlag = errors.New("the flag mission has no flag")
)

func (s Subscriber) Handle(ctx context.Context, event *missionsv1.FlagMissionDone) error {
	if err := event.GetDoneAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}
	if event.GetFlag() == "" {
		return errNoFlag
	}
	id, err := announcements.FlagMissionIDOf(event.GetMission().GetId())
	if err != nil {
		return err //nolint:wrapcheck // FlagMissionIDOf named it.
	}

	announcement, err := missionOf(id, event).Announcement(event.GetDoneAt().AsTime())
	if err != nil {
		return err //nolint:wrapcheck // Announcement named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announcement) //nolint:wrapcheck // the use case named it.
}

// A kind the chat does not know is still told, as a mission done.
func missionOf(id announcements.FlagMissionID, event *missionsv1.FlagMissionDone) announcements.FlagMission {
	mission := event.GetMission()
	done := announcements.FlagMissionOf(id, event.GetFlag(), mission.GetPoints())
	switch kind := mission.GetKind().(type) {
	case *missionsv1.FlagMission_TakeIn:
		return done.TakingIn(kind.TakeIn.GetCountryId(), kind.TakeIn.GetGoal())
	case *missionsv1.FlagMission_BreakShields:
		return done.BreakingShieldsOf(kind.BreakShields.GetCountryId(), kind.BreakShields.GetGoal())
	}
	return done
}
