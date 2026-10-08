package watch_lead_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Shares interface {
	Shares(ctx context.Context) (lead.Shares, error)
}

type Publisher interface {
	Publish(event proto.Message)
}

type Executor interface {
	Execute(ctx context.Context) error
}

// A boot this long after a season's end says nothing of it: the winner was told before the restart.
const endTold = 10 * time.Minute

func New(seasons calendar.Calendar, config lead.Config, clock cptime.Clock, shares Shares, publisher Publisher) *UseCase {
	return &UseCase{
		seasons:   seasons,
		config:    config,
		clock:     clock,
		shares:    shares,
		publisher: publisher,
		race:      lead.NewRace(config),
		told:      cpcolls.NewSet[calendar.Number](),
	}
}

type UseCase struct {
	seasons   calendar.Calendar
	config    lead.Config
	clock     cptime.Clock
	shares    Shares
	publisher Publisher

	race lead.Race
	told *cpcolls.Set[calendar.Number]
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) error {
	now := u.clock.Now()

	if phase := finale.PhaseAt(u.seasons, now); phase.Running() {
		return u.follow(ctx, phase.Season(), now)
	}

	u.race = lead.NewRace(u.config)

	season, ended := u.seasons.LastEnded(now)
	if !ended || u.told.Contains(season.Number) {
		return nil
	}
	if now.Sub(season.EndsAt) > endTold {
		u.told.Add(season.Number)
		return nil
	}

	return u.end(ctx, season)
}

func (u *UseCase) follow(ctx context.Context, season calendar.Season, now time.Time) error {
	shares, err := u.shares.Shares(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the shares during the finale: %w", err)
	}

	race, pass, passed := u.race.Next(shares, now)
	u.race = race
	if passed {
		u.publisher.Publish(&seasonsv1.LeadChanged{
			Season:    uint32(season.Number),
			Leader:    pass.Leader(),
			Passed:    pass.Passed(),
			ChangedAt: timestamppb.New(now),
		})
	}

	return nil
}

func (u *UseCase) end(ctx context.Context, season calendar.Season) error {
	shares, err := u.shares.Shares(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the shares at the end of season %d: %w", season.Number, err)
	}

	u.told.Add(season.Number)
	if winner, ok := shares.Leader(); ok {
		u.publisher.Publish(&seasonsv1.SeasonEnded{
			Season:  uint32(season.Number),
			Winner:  winner,
			EndedAt: timestamppb.New(season.EndsAt),
		})
	}

	return nil
}
