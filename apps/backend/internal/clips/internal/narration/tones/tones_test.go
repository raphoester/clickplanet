package tones_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration/tones"
)

func TestEveryToneIsABriefByName(t *testing.T) {
	all, err := tones.All()
	require.NoError(t, err)

	names := make([]narration.ToneName, 0, len(all))
	for _, tone := range all {
		names = append(names, tone.Name)
		assert.True(t, strings.HasPrefix(tone.Brief, "# "), "%s opens on its title", tone.Name)
	}
	assert.Equal(t, []narration.ToneName{"commentator", "correspondent", "deadpan", "documentary", "storyteller"}, names)
}

func TestTheTonesNamedComeInTheirOrderAndAnUnknownOneIsAnError(t *testing.T) {
	named, err := tones.Named("deadpan", "correspondent")
	require.NoError(t, err)
	assert.Equal(t, narration.ToneName("deadpan"), named[0].Name)
	assert.Equal(t, narration.ToneName("correspondent"), named[1].Name)

	_, err = tones.Named("shouty")
	assert.Error(t, err)
}
