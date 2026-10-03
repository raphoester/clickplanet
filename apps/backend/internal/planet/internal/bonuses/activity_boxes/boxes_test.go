package activity_boxes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/activity_boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestEachHookRecordsItsKindAgainstTheScope(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	recorded := &activity.Recorded{}
	boxes := activity_boxes.New(recorded, cptime.NewFixedClock(now))

	boxes.Offered("2001:db8::/64")
	boxes.Caught("2001:db8::/64", 1200*time.Millisecond)
	boxes.Lapsed("2001:db8::/64")
	boxes.Foreign("2001:db8:1::/64")

	caller := activity.CallerOfScope("2001:db8::/64")
	assert.Equal(t, []activity.Event{
		{At: now, Kind: activity.KindBoxOffered, Caller: caller},
		{At: now, Kind: activity.KindBoxCaught, Caller: caller, Delay: 1200 * time.Millisecond},
		{At: now, Kind: activity.KindBoxLapsed, Caller: caller},
		{At: now, Kind: activity.KindBoxForeign, Caller: activity.CallerOfScope("2001:db8:1::/64")},
	}, recorded.Events())
}
