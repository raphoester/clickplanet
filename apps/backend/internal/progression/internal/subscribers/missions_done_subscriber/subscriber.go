package missions_done_subscriber

import (
	"context"
	"errors"
	"fmt"
	"strings"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in record_stats_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*missionsv1.MissionsDone] = Subscriber{}

var ErrUnknownKind = errors.New("a kind of mission this build does not know")

func (s Subscriber) Handle(ctx context.Context, event *missionsv1.MissionsDone) error {
	account, err := accounts.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("missions done: %w", err)
	}
	tallies, err := talliesOf(event.GetByKind())
	if err != nil {
		return fmt.Errorf("missions done: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_stats_usecase.In{Account: account, Change: stats.MissionsDone(tallies)}) //nolint:wrapcheck // the use case named it.
}

func talliesOf(byKind []*missionsv1.KindDone) (map[stats.MissionKind]stats.MissionTally, error) {
	tallies := make(map[stats.MissionKind]stats.MissionTally, len(byKind))
	for _, done := range byKind {
		if done.GetKind() == missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED {
			continue
		}
		name, known := missionsv1.MissionKind_name[int32(done.GetKind())]
		if !known {
			return nil, fmt.Errorf("%w: %d", ErrUnknownKind, done.GetKind())
		}
		kind := stats.MissionKind(strings.ToLower(strings.TrimPrefix(name, "MISSION_KIND_")))
		tallies[kind] = stats.MissionTally{Done: done.GetDone(), Goals: done.GetGoals()}
	}
	return tallies, nil
}
