package posts

import (
	"slices"
	"time"
)

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

func (r Report) Complete() bool {
	return !slices.ContainsFunc(r, func(outcome Outcome) bool { return outcome.Failure != "" })
}
