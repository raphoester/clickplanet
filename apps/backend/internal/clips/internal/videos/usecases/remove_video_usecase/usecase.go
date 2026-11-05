package remove_video_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
)

type Files interface {
	Delete(ctx context.Context, id videos.VideoID) error
}

type Storage interface {
	Remove(ctx context.Context, id videos.VideoID) error
}

const writeTimeout = 10 * time.Second

func New(files Files, storage Storage) *UseCase {
	return &UseCase{files: files, storage: storage}
}

type UseCase struct {
	files   Files
	storage Storage
}

// The file goes first: a removal that fails halfway leaves the video listed, and asked again it is removed.
func (u *UseCase) Execute(ctx context.Context, id videos.VideoID) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := u.files.Delete(ctx, id); err != nil {
		return fmt.Errorf("cannot delete the file of the video: %w", err)
	}
	if err := u.storage.Remove(ctx, id); err != nil {
		return fmt.Errorf("cannot remove the video: %w", err)
	}
	return nil
}
