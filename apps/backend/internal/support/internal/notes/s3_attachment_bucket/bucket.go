package s3_attachment_bucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

const readsAtOnce = 8

type Lifetimes struct {
	Upload time.Duration
	View   time.Duration
}

// An empty config builds a bucket every link of which is cps3.ErrNoBucket: the module that takes no attachment asks
// for none.
func New(config cps3.Config, lifetimes Lifetimes, clock cptime.Clock, httpClient *http.Client) (*Bucket, error) {
	files, err := cps3.New(config, clock, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to build the bucket: %w", err)
	}
	return &Bucket{files: files, lifetimes: lifetimes}, nil
}

type Bucket struct {
	files     *cps3.Bucket
	lifetimes Lifetimes
}

func (b *Bucket) UploadURL(file notes.Attachment) (string, error) {
	link, err := b.files.UploadLink(file.ID.String(), cps3.Object{ContentType: string(file.MediaType), Size: file.Size},
		b.lifetimes.Upload)
	if err != nil {
		return "", fmt.Errorf("failed to sign the upload of the file %s: %w", file.ID, err)
	}
	return link, nil
}

func (b *Bucket) ViewURL(attachment string) (string, error) {
	link, err := b.files.ViewLink(attachment, b.lifetimes.View)
	if err != nil {
		return "", fmt.Errorf("failed to sign the view of the file %s: %w", attachment, err)
	}
	return link, nil
}

// What the bucket holds under each id, in their order: the type and the size the upload link signed.
func (b *Bucket) Attachments(ctx context.Context, ids ...notes.AttachmentID) ([]notes.Attachment, error) {
	files := make([]notes.Attachment, len(ids))
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(readsAtOnce)

	for n, id := range ids {
		group.Go(func() error {
			held, err := b.files.Head(ctx, id.String())
			switch {
			case errors.Is(err, cps3.ErrNotFound):
				return fmt.Errorf("%w: %s", notes.ErrNotUploaded, id)
			case err != nil:
				return fmt.Errorf("failed to read the file %s: %w", id, err)
			}
			files[n] = notes.Attachment{ID: id, MediaType: notes.MediaType(held.ContentType), Size: held.Size}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each read named what failed.
	}
	return files, nil
}

// Deletes the files, a few at once. A file the bucket does not hold is deleted already.
func (b *Bucket) Delete(ctx context.Context, ids ...notes.AttachmentID) error {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(readsAtOnce)

	for _, id := range ids {
		group.Go(func() error {
			if err := b.files.Delete(ctx, id.String()); err != nil {
				return fmt.Errorf("failed to delete the file %s: %w", id, err)
			}
			return nil
		})
	}
	return group.Wait() //nolint:wrapcheck // each deletion named what failed.
}
