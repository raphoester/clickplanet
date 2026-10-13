package posts

import (
	"errors"
	"fmt"
)

type VideoFile string

var ErrInvalidClip = errors.New("the clip is invalid")

type Clip struct {
	pick    Pick
	title   string
	caption string
	video   VideoFile
}

func (c Clip) TakeSnapshot() ClipSnapshot {
	return ClipSnapshot{Pick: c.pick, Title: c.title, Caption: c.caption, Video: c.video}
}

type ClipSnapshot struct {
	Pick  Pick
	Title string
	// Ready to post: its tags are in it.
	Caption string
	Video   VideoFile
}

func (s ClipSnapshot) Restore() (Clip, error) {
	switch {
	case s.Pick < 1:
		return Clip{}, fmt.Errorf("%w: no pick", ErrInvalidClip)
	case s.Title == "":
		return Clip{}, fmt.Errorf("%w: no title", ErrInvalidClip)
	case s.Caption == "":
		return Clip{}, fmt.Errorf("%w: no caption", ErrInvalidClip)
	case s.Video == "":
		return Clip{}, fmt.Errorf("%w: no video", ErrInvalidClip)
	}
	return Clip{pick: s.Pick, title: s.Title, caption: s.Caption, video: s.Video}, nil
}
