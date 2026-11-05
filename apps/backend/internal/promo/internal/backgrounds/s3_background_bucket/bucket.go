package s3_background_bucket

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
)

// Beside the support module's files and the clips' videos, in the same bucket. Cloudflare serves this prefix alone on
// the public domain.
const prefix = "promo/backgrounds/"

// publicBase is the public domain's origin, such as https://media.clickplanet.lol: a background is read from it, never
// from a signed link, so the browser and Cloudflare keep it.
func New(files *cps3.Bucket, uploadTTL time.Duration, publicBase string) *Bucket {
	return &Bucket{files: files, uploadTTL: uploadTTL, publicBase: strings.TrimRight(publicBase, "/")}
}

type Bucket struct {
	files      *cps3.Bucket
	uploadTTL  time.Duration
	publicBase string
}

func keyOf(id backgrounds.BackgroundID) string { return prefix + id.String() }

func off(err error) error {
	if errors.Is(err, cps3.ErrNoBucket) {
		return backgrounds.ErrBackgroundsOff
	}
	return err
}

func (b *Bucket) UploadURL(file backgrounds.File) (string, error) {
	link, err := b.files.UploadLink(keyOf(file.ID), cps3.Object{ContentType: string(file.MediaType), Size: file.Size}, b.uploadTTL)
	if err != nil {
		return "", fmt.Errorf("failed to sign the upload of the background %s: %w", file.ID, off(err))
	}
	return link, nil
}

// A file's id is drawn once and never names another, so its URL can be kept for good.
func (b *Bucket) PublicURL(id backgrounds.BackgroundID) string {
	return b.publicBase + "/" + keyOf(id)
}

func (b *Bucket) Held(ctx context.Context, id backgrounds.BackgroundID) (backgrounds.File, error) {
	held, err := b.files.Head(ctx, keyOf(id))
	switch {
	case errors.Is(err, cps3.ErrNotFound):
		return backgrounds.File{}, fmt.Errorf("%w: %s", backgrounds.ErrNotUploaded, id)
	case err != nil:
		return backgrounds.File{}, fmt.Errorf("failed to read the background %s: %w", id, off(err))
	}
	return backgrounds.File{ID: id, MediaType: backgrounds.MediaType(held.ContentType), Size: held.Size}, nil
}

func (b *Bucket) Delete(ctx context.Context, id backgrounds.BackgroundID) error {
	if err := b.files.Delete(ctx, keyOf(id)); err != nil {
		return fmt.Errorf("failed to delete the background %s: %w", id, off(err))
	}
	return nil
}
