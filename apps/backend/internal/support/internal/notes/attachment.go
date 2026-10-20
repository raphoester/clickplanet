package notes

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type MediaType string

const (
	MediaJPEG      MediaType = "image/jpeg"
	MediaPNG       MediaType = "image/png"
	MediaWebP      MediaType = "image/webp"
	MediaGIF       MediaType = "image/gif"
	MediaHEIC      MediaType = "image/heic"
	MediaHEIF      MediaType = "image/heif"
	MediaMP4       MediaType = "video/mp4"
	MediaQuickTime MediaType = "video/quicktime"
	MediaWebM      MediaType = "video/webm"
)

var mediaTypes = []MediaType{
	MediaJPEG, MediaPNG, MediaWebP, MediaGIF, MediaHEIC, MediaHEIF, MediaMP4, MediaQuickTime, MediaWebM,
}

func MediaTypes() []MediaType {
	return slices.Clone(mediaTypes)
}

func (m MediaType) known() bool {
	return slices.Contains(mediaTypes, m)
}

// Names a file in the bucket. Only its uploader is told it, so holding it is the proof of the upload.
type AttachmentID uuid.UUID

func AttachmentIDOf(value string) (AttachmentID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AttachmentID{}, fmt.Errorf("%w: %q is not an attachment id", ErrInvalidAttachment, value)
	}
	return AttachmentID(id), nil
}

func (id AttachmentID) String() string {
	return uuid.UUID(id).String()
}

type Attachment struct {
	ID        AttachmentID
	MediaType MediaType
	Size      int64
}

var (
	ErrInvalidAttachment = errors.New("the attachment is invalid")
	ErrNotUploaded       = errors.New("the file was not uploaded")
)

// The zero value takes no file.
type AttachmentRules struct {
	MaxPerNote int
	MaxBytes   int64
}

// The rules a page is told while the bucket has room bytes left: no file once it is full, none bigger than the room.
func (r AttachmentRules) Within(room int64) AttachmentRules {
	if room <= 0 {
		return AttachmentRules{}
	}
	return AttachmentRules{MaxPerNote: r.MaxPerNote, MaxBytes: min(r.MaxBytes, room)}
}

// Whether a file may be uploaded to be attached: the rules of one file, before any note.
func (r AttachmentRules) UploadError(file Attachment) error {
	if r.MaxPerNote == 0 {
		return fmt.Errorf("%w: this server takes no file", ErrInvalidAttachment)
	}
	if _, err := checked([]Attachment{file}); err != nil {
		return err
	}
	return r.refusal([]Attachment{file})
}

func (r AttachmentRules) refusal(files []Attachment) error {
	if len(files) > r.MaxPerNote {
		return fmt.Errorf("%w: %d files, and a note carries %d at most", ErrInvalidAttachment, len(files), r.MaxPerNote)
	}
	for _, file := range files {
		if file.Size > r.MaxBytes {
			return fmt.Errorf("%w: %d bytes, and a file is %d at most", ErrInvalidAttachment, file.Size, r.MaxBytes)
		}
	}
	return nil
}

func checked(files []Attachment) ([]Attachment, error) {
	if len(files) == 0 {
		return nil, nil
	}
	seen := cpcolls.NewSetWithCapacity[AttachmentID](len(files))
	for _, file := range files {
		switch {
		case file.ID == AttachmentID{}:
			return nil, fmt.Errorf("%w: no id", ErrInvalidAttachment)
		case seen.Contains(file.ID):
			return nil, fmt.Errorf("%w: %s twice", ErrInvalidAttachment, file.ID)
		case !file.MediaType.known():
			return nil, fmt.Errorf("%w: %q is not a kind of file a note carries", ErrInvalidAttachment, string(file.MediaType))
		case file.Size <= 0:
			return nil, fmt.Errorf("%w: no size", ErrInvalidAttachment)
		}
		seen.Add(file.ID)
	}
	return slices.Clone(files), nil
}
