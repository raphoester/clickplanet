package announce_champion_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

type Results interface {
	Table(ctx context.Context, book announcements.BookID) ([]announcements.Place, error)
}

type Announcer interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

type In struct {
	Season uint32
	Book   announcements.BookID
	At     time.Time
}

func New(results Results, announcer Announcer) *UseCase {
	return &UseCase{results: results, announcer: announcer}
}

type UseCase struct {
	results   Results
	announcer Announcer
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	table, err := u.results.Table(ctx, in.Book)
	if err != nil {
		return fmt.Errorf("failed to read the table of book %q: %w", in.Book, err)
	}

	champion := announcements.ChampionOf(in.Season, table)
	if !champion.Crowned() {
		return nil
	}

	announcement, err := champion.Announcement(in.At)
	if err != nil {
		return fmt.Errorf("cannot announce the champion of season %d: %w", in.Season, err)
	}

	return u.announcer.Execute(ctx, announcement)
}
