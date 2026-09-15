package cohort

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestMembersSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{}, clock, nil)
	for range 30 {
		clock.Advance(time.Second)
		for _, scope := range []string{"203.0.113.7", "203.0.113.9"} {
			w.Watch(detect.Click{Scope: scope, Tile: 1, Country: "BG", At: clock.Now()})
		}
	}

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(Config{}, clock, nil)
	require.NoError(t, restarted.Load(data))

	clock.Advance(time.Second)
	next := detect.Click{Scope: "203.0.113.7", Tile: 1, Country: "BG", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Suspect, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
	assert.Equal(t, w.prefixes, restarted.prefixes)
}

func TestForgetDropsMembersAndTheirIndexes(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{}, clock, nil)
	w.Attempted(detect.Click{Scope: "203.0.113.7", Country: "BG", At: clock.Now()})

	w.Forget(clock.Now().Add(time.Second))

	assert.Empty(t, w.members)
	assert.Empty(t, w.starts)
	assert.Empty(t, w.prefixes)
}
