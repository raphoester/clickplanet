package posts

import (
	"slices"
	"time"
)

type Outcome struct {
	Title     string
	At        time.Time
	Post      PostID
	Platforms []PlatformOutcome
}

type PlatformOutcome struct {
	Platform Platform
	Failure  string
}

type Report []Outcome

// failure: the post was made on no platform.
func (r Report) With(post Post, scheduled Scheduled, failure error) Report {
	outcome := Outcome{Title: post.clip.title, At: post.at, Post: scheduled.ID}
	if failure != nil {
		outcome.Post = ""
	}
	for _, platform := range post.platforms {
		refusal := scheduled.Refused[platform]
		if failure != nil {
			refusal = failure
		}
		outcome.Platforms = append(outcome.Platforms, PlatformOutcome{Platform: platform, Failure: messageOf(refusal)})
	}
	return append(slices.Clone(r), outcome)
}

func (r Report) Complete() bool {
	return !slices.ContainsFunc(r, func(outcome Outcome) bool {
		return outcome.Post == "" || slices.ContainsFunc(outcome.Platforms, func(platform PlatformOutcome) bool { return platform.Failure != "" })
	})
}

func messageOf(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
