package posts

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

var ErrUnsent = errors.New("a clip was not scheduled")

type Outcome struct {
	Title   string
	At      time.Time
	Post    PostID
	Failure string
}

type Report []Outcome

func (r Report) With(post Post, id PostID, failure error) Report {
	outcome := Outcome{Title: post.clip.title, At: post.at, Post: id}
	if failure != nil {
		outcome.Post = ""
		outcome.Failure = failure.Error()
	}
	return append(slices.Clone(r), outcome)
}

func (r Report) UnsentError() error {
	unsent := 0
	for _, outcome := range r {
		if outcome.Failure != "" {
			unsent++
		}
	}
	if unsent == 0 {
		return nil
	}
	return fmt.Errorf("%w: %d of %d", ErrUnsent, unsent, len(r))
}
