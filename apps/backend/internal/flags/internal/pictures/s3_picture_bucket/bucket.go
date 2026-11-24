package s3_picture_bucket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
)

// A picture's key is its content's hash: it never holds another picture.
const kept = "public, max-age=31536000, immutable"

func New(files *cps3.Bucket, uploadTTL time.Duration) *Bucket {
	return &Bucket{files: files, uploadTTL: uploadTTL}
}

type Bucket struct {
	files     *cps3.Bucket
	uploadTTL time.Duration
}

var _ pictures.Bucket = (*Bucket)(nil)

func off(err error) error {
	if errors.Is(err, cps3.ErrNoBucket) {
		return fmt.Errorf("%w: %w", pictures.ErrOff, err)
	}
	return err
}

func (b *Bucket) UploadURL(file pictures.File) (string, error) {
	link, err := b.files.UploadLink(file.ID.Key(), cps3.Object{ContentType: file.ContentType, Size: file.Size}, b.uploadTTL)
	if err != nil {
		return "", off(err)
	}
	return link, nil
}

func (b *Bucket) Uploaded(ctx context.Context, id pictures.UploadID, most int64) ([]byte, error) {
	file, err := b.files.Read(ctx, id.Key(), most)
	switch {
	case errors.Is(err, cps3.ErrNotFound):
		return nil, fmt.Errorf("%w: %s", pictures.ErrUnknownUpload, id)
	case errors.Is(err, cps3.ErrTooLarge):
		return nil, fmt.Errorf("%w: %w", pictures.ErrTooLarge, err)
	case err != nil:
		return nil, off(err)
	}
	return file, nil
}

func (b *Bucket) Publish(ctx context.Context, drawn pictures.Drawn) error {
	return off(b.files.Write(ctx, drawn.Picture.Key(), "image/png", kept, drawn.PNG))
}

func (b *Bucket) Discard(ctx context.Context, id pictures.UploadID) error {
	return off(b.files.Delete(ctx, id.Key()))
}

// A picture held for review is read through a link the backoffice asks for, never kept by a browser.
const held = "private, no-store"

func (b *Bucket) Hold(ctx context.Context, key string, png []byte) error {
	return off(b.files.Write(ctx, key, "image/png", held, png))
}

func (b *Bucket) Held(ctx context.Context, key string, most int64) ([]byte, error) {
	file, err := b.files.Read(ctx, key, most)
	switch {
	case errors.Is(err, cps3.ErrNotFound):
		return nil, fmt.Errorf("%w: %s", pictures.ErrUnknownHeld, key)
	case err != nil:
		return nil, off(err)
	}
	return file, nil
}

func (b *Bucket) Drop(ctx context.Context, key string) error {
	return off(b.files.Delete(ctx, key))
}

// HeldURL is a link to see a picture held in private, good for the time given.
func (b *Bucket) HeldURL(key string, ttl time.Duration) (string, error) {
	link, err := b.files.ViewLink(key, ttl)
	if err != nil {
		return "", off(err)
	}
	return link, nil
}
