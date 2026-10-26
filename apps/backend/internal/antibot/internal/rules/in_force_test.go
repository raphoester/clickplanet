package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestWhatIsReadIsTheSetEnforcedLast(t *testing.T) {
	inForce := rules.NewInForce()
	for _, kept := range []rules.Snapshot{
		rules.Seeded(),
		rules.Named("rush", func(s *rules.Snapshot) { s.Cohort.CertainMembers = 50 }),
	} {
		set, err := kept.Restore()
		require.NoError(t, err)
		inForce.Enforce(set)
	}

	assert.Equal(t, rules.SetName("rush"), inForce.Name())
	assert.Equal(t, 50, inForce.Cohort().CertainMembers)
	assert.Equal(t, rules.Seeded().Bans, inForce.Bans())
	assert.Equal(t, rules.Seeded().Jury, inForce.Jury())
	assert.Equal(t, rules.Seeded().Retaker, inForce.Retaker())
	assert.Equal(t, rules.Seeded().Sequencer, inForce.Sequencer())
	assert.Equal(t, rules.Seeded().Metronome, inForce.Metronome())
	assert.Equal(t, rules.Seeded().Defender, inForce.Defender())
	assert.Equal(t, rules.Seeded().Catcher, inForce.Catcher())
	assert.Equal(t, rules.Seeded().Scraper, inForce.Scraper())
	assert.Equal(t, rules.Seeded().Churner, inForce.Churner())
	assert.Equal(t, rules.Seeded().Hopper, inForce.Hopper())
	assert.Equal(t, rules.Seeded().Rotator, inForce.Rotator())
}
