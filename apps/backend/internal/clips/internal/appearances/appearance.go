package appearances

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

// The posting service's id of a post: one clip on every platform it went to.
type PostID string

var ErrNoPost = errors.New("an appearance names no post")

func PostIDOf(value string) (PostID, error) {
	if value == "" {
		return "", ErrNoPost
	}
	return PostID(value), nil
}

type Platform string

const (
	TikTok  Platform = "tiktok"
	YouTube Platform = "youtube"
)

var ErrUnknownPlatform = errors.New("not a platform the clips go to")

func PlatformOf(name string) (Platform, error) {
	switch Platform(name) {
	case TikTok, YouTube:
		return Platform(name), nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownPlatform, name)
}

// A clip out on some platforms, at a time.
type Post struct {
	ID          PostID
	Title       string
	Platforms   []Platform
	ScheduledAt time.Time
}

var ErrNowhere = errors.New("a post on no platform names no one")

func PostOf(id PostID, title string, platforms []Platform, scheduledAt time.Time) (Post, error) {
	if len(platforms) == 0 {
		return Post{}, ErrNowhere
	}
	if scheduledAt.IsZero() {
		return Post{}, errors.New("a post with no time")
	}
	return Post{ID: id, Title: title, Platforms: slices.Clone(platforms), ScheduledAt: scheduledAt.UTC()}, nil
}

// A post that names a player.
type Appearance struct {
	Post    Post
	Account casts.AccountID
}

func (p Post) AppearancesOf(accounts []casts.AccountID) []Appearance {
	named := make([]Appearance, 0, len(accounts))
	for _, account := range accounts {
		named = append(named, Appearance{Post: p, Account: account})
	}
	return named
}

type Key struct {
	Post    PostID
	Account casts.AccountID
}

// The game tells a player of a post for a week.
const shownFor = 7 * 24 * time.Hour

func ShownSince(now time.Time) time.Time {
	return now.Add(-shownFor)
}
