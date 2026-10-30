package videos

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

type VideoID uuid.UUID

var ErrInvalidVideoID = errors.New("not a video id")

func VideoIDOf(value string) (VideoID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return VideoID{}, fmt.Errorf("%w: %q", ErrInvalidVideoID, value)
	}
	return VideoID(id), nil
}

func (v VideoID) String() string { return uuid.UUID(v).String() }

type IDProvider interface {
	NewID() (VideoID, error)
}

type MediaType string

var extensions = map[MediaType]string{"video/mp4": ".mp4", "video/quicktime": ".mov", "video/webm": ".webm"}

var (
	ErrInvalidVideo = errors.New("not a video the operator may keep")
	ErrNotUploaded  = errors.New("the bucket holds no such video")
	ErrNoVideo      = errors.New("no video of that id")
	ErrVideosOff    = errors.New("videos are off on this server: clips.videos.enabled")
)

// A file the page may send, or one the bucket holds: its type and its size.
type File struct {
	ID        VideoID
	MediaType MediaType
	Size      int64
}

type Rules struct {
	MaxBytes int64
}

func (r Rules) FileError(file File) error {
	if _, ok := extensions[file.MediaType]; !ok {
		return fmt.Errorf("%w: %q is not an mp4, a mov or a webm", ErrInvalidVideo, file.MediaType)
	}
	if file.Size <= 0 || file.Size > r.MaxBytes {
		return fmt.Errorf("%w: %d bytes, and a video takes 1 to %d", ErrInvalidVideo, file.Size, r.MaxBytes)
	}
	return nil
}

const (
	maxName    = 100
	maxCaption = 2200
)

type Video struct {
	file    File
	name    string
	caption string
	keptAt  time.Time
}

type VideoSnapshot struct {
	File    File
	Name    string
	Caption string
	KeptAt  time.Time
}

// The file is the one the bucket holds, never what the page said it sent.
func Keep(held File, name, caption string, at time.Time, rules Rules) (Video, error) {
	if err := rules.FileError(held); err != nil {
		return Video{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxName {
		return Video{}, fmt.Errorf("%w: a name of 1 to %d characters", ErrInvalidVideo, maxName)
	}
	caption = strings.TrimSpace(caption)
	if utf8.RuneCountInString(caption) > maxCaption {
		return Video{}, fmt.Errorf("%w: a caption of %d characters at most", ErrInvalidVideo, maxCaption)
	}
	return Video{file: held, name: name, caption: caption, keptAt: at}, nil
}

func (s VideoSnapshot) Restore() Video {
	return Video{file: s.File, name: s.Name, caption: s.Caption, keptAt: s.KeptAt}
}

func (v Video) TakeSnapshot() VideoSnapshot {
	return VideoSnapshot{File: v.file, Name: v.name, Caption: v.caption, KeptAt: v.keptAt}
}

// The name a download is saved under: letters, digits, - and _, and the type's extension.
func FileName(name string, media MediaType) string {
	var out strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)), r == '-', r == '_':
			out.WriteRune(r)
		case r == ' ' || r == '.':
			out.WriteRune('-')
		}
	}
	base := strings.Trim(out.String(), "-")
	if base == "" {
		base = "video"
	}
	return base + extensions[media]
}
