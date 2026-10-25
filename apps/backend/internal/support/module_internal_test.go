package support

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAConfigThatSetsNoLimitTakesTheDefaultOnes(t *testing.T) {
	config := Config{}.withDefaults()

	assert.Equal(t, 3, config.RateLimiter.Burst, "three notes at once")
	assert.Equal(t, 10, config.NoteLimiter.Burst, "ten notes a day")
	assert.InDelta(t, 10.0/day.Seconds(), config.NoteLimiter.PerSecond, 1e-12)
	assert.Equal(t, 10, config.Attachments.MaxPerNote)
	assert.Equal(t, int64(50<<20), config.Attachments.MaxBytes)
}
