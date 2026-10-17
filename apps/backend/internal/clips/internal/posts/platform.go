package posts

import (
	"errors"
	"fmt"
	"slices"
)

type Platform string

const (
	TikTok  Platform = "tiktok"
	YouTube Platform = "youtube"
)

var ErrInvalidPlatforms = errors.New("the platforms are invalid")

// Each once, and at least one.
func PlatformsOf(names ...string) ([]Platform, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("%w: none", ErrInvalidPlatforms)
	}

	platforms := make([]Platform, 0, len(names))
	for _, name := range names {
		platform := Platform(name)
		if !platform.known() {
			return nil, fmt.Errorf("%w: no platform is called %q", ErrInvalidPlatforms, name)
		}
		if slices.Contains(platforms, platform) {
			return nil, fmt.Errorf("%w: %s twice", ErrInvalidPlatforms, platform)
		}
		platforms = append(platforms, platform)
	}
	return platforms, nil
}

func (p Platform) known() bool {
	switch p {
	case TikTok, YouTube:
		return true
	default:
		return false
	}
}
