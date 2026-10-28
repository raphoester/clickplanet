package posts

import (
	"slices"
	"time"
)

type Outcome struct {
	Title string
	// The arm of each experiment the clip is in.
	Variants  Variants
	At        time.Time
	Post      PostID
	Platforms []PlatformOutcome
	// The trending track attached on TikTok; nil for none.
	Sound *Sound
	// Why the clip went out with its anthem though drawn for a trending track; empty when it did not fall back.
	SoundMissed string
	// Not sent: every platform's day was full. Not a failure: its story is still untold, for a later run.
	Held bool
	// The players it names, by their number in its cast.
	Cast     string
	Fighters []FighterNumber
}

type PlatformOutcome struct {
	Platform Platform
	Failure  string
}

type Report []Outcome

// failure: the post was made on no platform.
func (r Report) With(post Post, scheduled Scheduled, failure error) Report {
	outcome := Outcome{
		Title: post.clip.title, Variants: post.variants(), At: post.at, Post: scheduled.ID, Sound: post.sound,
		SoundMissed: messageOf(post.missed), Cast: post.clip.cast, Fighters: post.clip.fighterNumbers(),
	}
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

func (r Report) WithHeld(clip Clip) Report {
	return append(slices.Clone(r), Outcome{Title: clip.title, Variants: clip.variants, Held: true})
}

// A clip held for a full day is not missing.
func (r Report) Complete() bool {
	return !slices.ContainsFunc(r, func(outcome Outcome) bool {
		return !outcome.Held && (outcome.Post == "" ||
			slices.ContainsFunc(outcome.Platforms, func(platform PlatformOutcome) bool { return platform.Failure != "" }))
	})
}

func messageOf(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// A post that names players of a cast, on the platforms that took it: the server tells each in the game once it is out.
type Appearance struct {
	Cast      string
	Fighters  []FighterNumber
	Post      PostID
	Title     string
	Platforms []Platform
	At        time.Time
}

// The posts made that name someone.
func (r Report) Appearances() []Appearance {
	var named []Appearance
	for _, outcome := range r {
		if outcome.Held || outcome.Post == "" || len(outcome.Fighters) == 0 {
			continue
		}
		appearance := Appearance{
			Cast: outcome.Cast, Fighters: outcome.Fighters, Post: outcome.Post, Title: outcome.Title, At: outcome.At,
		}
		for _, platform := range outcome.Platforms {
			if platform.Failure == "" {
				appearance.Platforms = append(appearance.Platforms, platform.Platform)
			}
		}
		if len(appearance.Platforms) > 0 {
			named = append(named, appearance)
		}
	}
	return named
}
