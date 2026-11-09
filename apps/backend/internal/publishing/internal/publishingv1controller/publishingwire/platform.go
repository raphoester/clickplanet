// The wire's words for the publishing domain's.
package publishingwire

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
)

var ErrNoPlatform = errors.New("no platform")

// Each known, each once, at least one: the domain's.
func Platforms(wire []publishingv1.Platform) ([]calendar.Platform, error) {
	if len(wire) == 0 {
		return nil, ErrNoPlatform
	}
	names := make([]string, 0, len(wire))
	for _, platform := range wire {
		switch platform {
		case publishingv1.Platform_PLATFORM_TIKTOK:
			names = append(names, string(calendar.TikTok))
		case publishingv1.Platform_PLATFORM_YOUTUBE:
			names = append(names, string(calendar.YouTube))
		default:
			return nil, fmt.Errorf("%w %s", calendar.ErrUnknownPlatform, platform)
		}
	}
	return calendar.PlatformsOf(names...) //nolint:wrapcheck // the domain already says which platform is wrong.
}

func Platform(platform calendar.Platform) publishingv1.Platform {
	switch platform {
	case calendar.TikTok:
		return publishingv1.Platform_PLATFORM_TIKTOK
	case calendar.YouTube:
		return publishingv1.Platform_PLATFORM_YOUTUBE
	default:
		return publishingv1.Platform_PLATFORM_UNSPECIFIED
	}
}

func Slot(slot calendar.Slot) *publishingv1.Slot {
	platforms := make([]publishingv1.Platform, 0, len(slot.Platforms))
	for _, platform := range slot.Platforms {
		platforms = append(platforms, Platform(platform))
	}
	return &publishingv1.Slot{At: timestamppb.New(slot.At), Platforms: platforms}
}
