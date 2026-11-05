// Package backgrounds is the library of images a teaser's cards are drawn over: the operator uploads each one to the
// bucket, under promo/backgrounds/<id>, and the players load it from its public URL.
package backgrounds

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type BackgroundID uuid.UUID

var ErrInvalidBackgroundID = errors.New("not a background id")

func BackgroundIDOf(value string) (BackgroundID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return BackgroundID{}, fmt.Errorf("%w: %q", ErrInvalidBackgroundID, value)
	}
	return BackgroundID(id), nil
}

func (b BackgroundID) String() string { return uuid.UUID(b).String() }

type IDProvider interface {
	NewID() (BackgroundID, error)
}

type MediaType string

var mediaTypes = []MediaType{"image/jpeg", "image/png", "image/webp"}

var (
	ErrInvalidBackground = errors.New("not a background the operator may keep")
	ErrNotUploaded       = errors.New("the bucket holds no such background")
	ErrNoBackground      = errors.New("no background of that id")
	ErrShown             = errors.New("a card shows this background")
	ErrBackgroundsOff    = errors.New("backgrounds are off on this server: promo.backgrounds.enabled")
)

// A file the page may send, or one the bucket holds: its type and its size.
type File struct {
	ID        BackgroundID
	MediaType MediaType
	Size      int64
}

type Rules struct {
	MaxBytes int64
}

func (r Rules) FileError(file File) error {
	known := false
	for _, media := range mediaTypes {
		known = known || file.MediaType == media
	}
	if !known {
		return fmt.Errorf("%w: %q is not a JPEG, a PNG or a WebP", ErrInvalidBackground, file.MediaType)
	}
	if file.Size <= 0 || file.Size > r.MaxBytes {
		return fmt.Errorf("%w: %d bytes, and a background takes 1 to %d", ErrInvalidBackground, file.Size, r.MaxBytes)
	}
	return nil
}

const maxName = 100

type Background struct {
	file   File
	name   string
	keptAt time.Time
}

type BackgroundSnapshot struct {
	File   File
	Name   string
	KeptAt time.Time
}

// The file is the one the bucket holds, never what the page said it sent.
func Keep(held File, name string, at time.Time, rules Rules) (Background, error) {
	if err := rules.FileError(held); err != nil {
		return Background{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxName {
		return Background{}, fmt.Errorf("%w: a name of 1 to %d characters", ErrInvalidBackground, maxName)
	}
	return Background{file: held, name: name, keptAt: at}, nil
}

func (s BackgroundSnapshot) Restore() Background {
	return Background{file: s.File, name: s.Name, keptAt: s.KeptAt}
}

func (b Background) TakeSnapshot() BackgroundSnapshot {
	return BackgroundSnapshot{File: b.file, Name: b.name, KeptAt: b.keptAt}
}
