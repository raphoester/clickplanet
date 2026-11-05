package s3_video_bucket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
)

// Beside the support module's files, in the same bucket.
const prefix = "clips/videos/"

type Lifetimes struct {
	Upload time.Duration
	View   time.Duration
}

func New(files *cps3.Bucket, lifetimes Lifetimes) *Bucket {
	return &Bucket{files: files, lifetimes: lifetimes}
}

type Bucket struct {
	files     *cps3.Bucket
	lifetimes Lifetimes
}

func keyOf(id videos.VideoID) string { return prefix + id.String() }

func off(err error) error {
	if errors.Is(err, cps3.ErrNoBucket) {
		return videos.ErrVideosOff
	}
	return err
}

func (b *Bucket) UploadURL(file videos.File) (string, error) {
	link, err := b.files.UploadLink(keyOf(file.ID), cps3.Object{ContentType: string(file.MediaType), Size: file.Size},
		b.lifetimes.Upload)
	if err != nil {
		return "", fmt.Errorf("failed to sign the upload of the video %s: %w", file.ID, off(err))
	}
	return link, nil
}

func (b *Bucket) ViewURL(id videos.VideoID) (string, error) {
	link, err := b.files.ViewLink(keyOf(id), b.lifetimes.View)
	if err != nil {
		return "", fmt.Errorf("failed to sign the view of the video %s: %w", id, off(err))
	}
	return link, nil
}

func (b *Bucket) DownloadURL(id videos.VideoID, fileName string) (string, error) {
	link, err := b.files.DownloadLink(keyOf(id), fileName, b.lifetimes.View)
	if err != nil {
		return "", fmt.Errorf("failed to sign the download of the video %s: %w", id, off(err))
	}
	return link, nil
}

func (b *Bucket) Held(ctx context.Context, id videos.VideoID) (videos.File, error) {
	held, err := b.files.Head(ctx, keyOf(id))
	switch {
	case errors.Is(err, cps3.ErrNotFound):
		return videos.File{}, fmt.Errorf("%w: %s", videos.ErrNotUploaded, id)
	case err != nil:
		return videos.File{}, fmt.Errorf("failed to read the video %s: %w", id, off(err))
	}
	return videos.File{ID: id, MediaType: videos.MediaType(held.ContentType), Size: held.Size}, nil
}

func (b *Bucket) Delete(ctx context.Context, id videos.VideoID) error {
	if err := b.files.Delete(ctx, keyOf(id)); err != nil {
		return fmt.Errorf("failed to delete the video %s: %w", id, off(err))
	}
	return nil
}
