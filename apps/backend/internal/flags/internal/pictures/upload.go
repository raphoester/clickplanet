package pictures

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrUnknownUpload = errors.New("no picture was uploaded under this id")
	ErrTooLarge      = errors.New("the picture is larger than the upload allows")
	ErrOff           = errors.New("the flags' pictures are off")
)

type UploadID uuid.UUID

func ParseUploadID(s string) (UploadID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UploadID{}, fmt.Errorf("%w: %q", ErrUnknownUpload, s)
	}
	return UploadID(id), nil
}

func (id UploadID) String() string {
	return uuid.UUID(id).String()
}

// Key is where the bucket keeps the file uploaded: private, until it is drawn again.
func (id UploadID) Key() string {
	return "flags/uploads/" + id.String()
}

type IDProvider interface {
	NewID() (UploadID, error)
}

// File is what the page says it will upload.
type File struct {
	ID          UploadID
	ContentType string
	Size        int64
}

type Rules struct {
	MaxBytes int64
}

var contentTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true}

func (r Rules) Check(contentType string, size int64) error {
	switch {
	case !contentTypes[contentType]:
		return fmt.Errorf("%w: %q", ErrNotAPicture, contentType)
	case size <= 0 || size > r.MaxBytes:
		return fmt.Errorf("%w: %d bytes, at most %d", ErrTooLarge, size, r.MaxBytes)
	}
	return nil
}

type Bucket interface {
	UploadURL(file File) (string, error)
	// Uploaded is the file uploaded under the id: ErrUnknownUpload when there is none, ErrTooLarge past most bytes.
	Uploaded(ctx context.Context, id UploadID, most int64) ([]byte, error)
	Publish(ctx context.Context, drawn Drawn) error
	Discard(ctx context.Context, id UploadID) error
}

func NewGallery(bucket Bucket, rules Rules) Gallery {
	return Gallery{bucket: bucket, rules: rules}
}

// Gallery draws each picture uploaded again and makes it public.
type Gallery struct {
	bucket Bucket
	rules  Rules
}

func (g Gallery) Draw(ctx context.Context, id UploadID) (Picture, error) {
	file, err := g.bucket.Uploaded(ctx, id, g.rules.MaxBytes)
	if err != nil {
		return Picture{}, fmt.Errorf("failed to read the picture uploaded: %w", err)
	}
	drawn, err := Redraw(file)
	if err != nil {
		return Picture{}, err
	}
	if err := g.bucket.Publish(ctx, drawn); err != nil {
		return Picture{}, fmt.Errorf("failed to make the picture public: %w", err)
	}
	return drawn.Picture, nil
}

// Discard deletes the file uploaded, once the flag that wears its picture is kept.
func (g Gallery) Discard(ctx context.Context, id UploadID) error {
	if err := g.bucket.Discard(ctx, id); err != nil {
		return fmt.Errorf("failed to discard the file uploaded: %w", err)
	}
	return nil
}
