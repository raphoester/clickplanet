package notes

import (
	"errors"
	"fmt"
	"slices"
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

// Its place in a note's list names its file in the bucket: a list is never reordered.
type Attachment struct {
	MediaType MediaType
	Size      int64
}

var ErrInvalidAttachment = errors.New("the attachment is invalid")

// The zero value takes no file.
type AttachmentRules struct {
	MaxPerNote int
	MaxBytes   int64
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
	for _, file := range files {
		switch {
		case !file.MediaType.known():
			return nil, fmt.Errorf("%w: %q is not a kind of file a note carries", ErrInvalidAttachment, string(file.MediaType))
		case file.Size <= 0:
			return nil, fmt.Errorf("%w: no size", ErrInvalidAttachment)
		}
	}
	return slices.Clone(files), nil
}
