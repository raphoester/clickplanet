package announce_round_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

type Results interface {
	Round(ctx context.Context, round announcements.RoundID) ([]announcements.Place, error)
}

type Announcer interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

type In struct {
	Round   announcements.Round
	RoundID announcements.RoundID
	At      time.Time
}

func New(results Results, announcer Announcer) *UseCase {
	return &UseCase{results: results, announcer: announcer}
}

type UseCase struct {
	results   Results
	announcer Announcer
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	places, err := u.results.Round(ctx, in.RoundID)
	if err != nil {
		return fmt.Errorf("failed to read the results of round %q: %w", in.RoundID, err)
	}

	announcement, err := in.Round.Placed(places).Announcement(in.At)
	if err != nil {
		return fmt.Errorf("cannot announce round %q: %w", in.RoundID, err)
	}

	return u.announcer.Execute(ctx, announcement)
}
