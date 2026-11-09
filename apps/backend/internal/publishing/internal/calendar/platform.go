package calendar

import (
	"errors"
	"fmt"
	"slices"
)

// A platform the game posts on, by the name the posting service gives it.
type Platform string

const (
	TikTok  Platform = "tiktok"
	YouTube Platform = "youtube"
)

var ErrUnknownPlatform = errors.New("unknown platform")

// Each known, each once.
func PlatformsOf(names ...string) ([]Platform, error) {
	platforms := make([]Platform, 0, len(names))
	for _, name := range names {
		platform := Platform(name)
		if !platform.known() {
			return nil, fmt.Errorf("%w %q: only %s and %s", ErrUnknownPlatform, name, TikTok, YouTube)
		}
		if slices.Contains(platforms, platform) {
			return nil, fmt.Errorf("the platform %s is named twice", platform)
		}
		platforms = append(platforms, platform)
	}
	return platforms, nil
}

func (p Platform) known() bool {
	return p == TikTok || p == YouTube
}
