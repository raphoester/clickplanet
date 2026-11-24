package pictures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrUnknownUpload = errors.New("no picture was uploaded under this id")
	ErrTooLarge      = errors.New("the picture is larger than the upload allows")
	ErrOff           = errors.New("the flags' pictures are off")
	ErrUnknownHeld   = errors.New("no picture is held in private under this key")
	ErrNotThePicture = errors.New("the picture held is not the one drawn")
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
	// Hold keeps a picture drawn again in private, under a key of its own.
	Hold(ctx context.Context, key string, png []byte) error
	// Held is the picture kept in private under the key: ErrUnknownHeld when there is none.
	Held(ctx context.Context, key string, most int64) ([]byte, error)
	Drop(ctx context.Context, key string) error
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

// Redrawn is the file uploaded drawn again, made public nowhere.
func (g Gallery) Redrawn(ctx context.Context, id UploadID) (Drawn, error) {
	file, err := g.bucket.Uploaded(ctx, id, g.rules.MaxBytes)
	if err != nil {
		return Drawn{}, fmt.Errorf("failed to read the picture uploaded: %w", err)
	}
	return Redraw(file)
}

// Hold keeps a picture drawn again in private, until it is reviewed.
func (g Gallery) Hold(ctx context.Context, key string, drawn Drawn) error {
	if err := g.bucket.Hold(ctx, key, drawn.PNG); err != nil {
		return fmt.Errorf("failed to keep the picture in private: %w", err)
	}
	return nil
}

// Release makes public the picture held in private under the key, once it is checked to be the picture approved.
func (g Gallery) Release(ctx context.Context, key string, picture Picture) error {
	png, err := g.bucket.Held(ctx, key, g.rules.MaxBytes)
	if err != nil {
		return fmt.Errorf("failed to read the picture held: %w", err)
	}
	sum := sha256.Sum256(png)
	if hex.EncodeToString(sum[:]) != picture.hash {
		return fmt.Errorf("%w: %s", ErrNotThePicture, key)
	}
	if err := g.bucket.Publish(ctx, Drawn{PNG: png, Picture: picture}); err != nil {
		return fmt.Errorf("failed to make the picture public: %w", err)
	}
	return nil
}

// Drop deletes the picture held in private under the key.
func (g Gallery) Drop(ctx context.Context, key string) error {
	if err := g.bucket.Drop(ctx, key); err != nil {
		return fmt.Errorf("failed to delete the picture held: %w", err)
	}
	return nil
}

// Discard deletes the file uploaded, once the flag that wears its picture is kept.
func (g Gallery) Discard(ctx context.Context, id UploadID) error {
	if err := g.bucket.Discard(ctx, id); err != nil {
		return fmt.Errorf("failed to discard the file uploaded: %w", err)
	}
	return nil
}
