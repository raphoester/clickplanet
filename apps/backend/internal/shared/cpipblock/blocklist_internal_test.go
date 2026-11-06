package cpipblock

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASharedBlocklistIsParsedOnce(t *testing.T) {
	blocklist := New(Config{Enabled: true})

	require.NoError(t, blocklist.Load(), "the planet loads it")
	vpn := blocklist.sets[0].set
	require.NoError(t, blocklist.Load(), "then the chat")

	assert.Same(t, vpn, blocklist.sets[0].set, "the chat's load parses nothing again")
}

func TestASharedBlocklistThatFailedFailsEveryLoad(t *testing.T) {
	blocklist := New(Config{Enabled: true, Allow: []string{"10.0.0.1"}})

	first := blocklist.Load()
	require.Error(t, first)

	assert.Same(t, first, blocklist.Load(), "the second module is told the first parse's failure")
}
