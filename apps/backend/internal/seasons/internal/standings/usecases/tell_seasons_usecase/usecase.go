package tell_seasons_usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Rounds interface {
	ClosedIn(ctx context.Context, season calendar.Number) ([]rounds.Closed, error)
}

type Scores interface {
	Scores(ctx context.Context, season calendar.Number) ([]standings.Score, error)
}

type Publisher interface {
	Publish(event proto.Message)
}

type Told struct {
	Season  calendar.Number
	Players int
}

type Executor interface {
	Execute(ctx context.Context, told []calendar.Number) ([]Told, error)
}

const timeout = 30 * time.Second

func New(rounds Rounds, scores Scores, events Publisher, seasons calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{rounds: rounds, scores: scores, events: events, seasons: seasons, clock: clock}
}

type UseCase struct {
	rounds  Rounds
	scores  Scores
	events  Publisher
	seasons calendar.Calendar
	clock   cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, told []calendar.Number) ([]Told, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		now   []Told
		fails error
	)
	for _, season := range u.seasons.Ended(u.clock.Now()) {
		if slices.Contains(told, season.Number) {
			continue
		}
		players, over, err := u.tell(ctx, season.Number)
		if err != nil {
			fails = errors.Join(fails, err)
			continue
		}
		if over {
			now = append(now, Told{Season: season.Number, Players: players})
		}
	}
	return now, fails
}

func (u *UseCase) tell(ctx context.Context, season calendar.Number) (int, bool, error) {
	closed, err := u.rounds.ClosedIn(ctx, season)
	if err != nil {
		return 0, false, fmt.Errorf("failed to read the rounds season %d closed: %w", season, err)
	}
	end, over := rounds.EndOf(closed)
	if !over {
		return 0, false, nil
	}
	scores, err := u.scores.Scores(ctx, season)
	if err != nil {
		return 0, false, fmt.Errorf("failed to read the players of season %d: %w", season, err)
	}

	for _, score := range scores {
		u.events.Publish(&seasonsv1.SeasonPlayed{
			Season:    uint32(season),
			AccountId: score.Account.String(),
			MainFlag:  string(score.Main),
			Tiles:     score.Tiles,
			Won:       end.Won(rounds.Country(score.Main)),
		})
	}
	return len(scores), true, nil
}
