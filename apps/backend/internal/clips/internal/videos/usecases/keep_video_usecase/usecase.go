package keep_video_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Files interface {
	Held(ctx context.Context, id videos.VideoID) (videos.File, error)
}

type Storage interface {
	Keep(ctx context.Context, video videos.Video) error
}

type In struct {
	ID      videos.VideoID
	Name    string
	Caption string
}

const readTimeout = 10 * time.Second

func New(files Files, storage Storage, rules videos.Rules, clock cptime.Clock) *UseCase {
	return &UseCase{files: files, storage: storage, rules: rules, clock: clock}
}

type UseCase struct {
	files   Files
	storage Storage
	rules   videos.Rules
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	held, err := u.files.Held(ctx, in.ID)
	if err != nil {
		return fmt.Errorf("cannot read the video sent: %w", err)
	}
	video, err := videos.Keep(held, in.Name, in.Caption, u.clock.Now(), u.rules)
	if err != nil {
		return fmt.Errorf("cannot keep the video: %w", err)
	}
	if err := u.storage.Keep(ctx, video); err != nil {
		return fmt.Errorf("failed to keep the video: %w", err)
	}
	return nil
}
