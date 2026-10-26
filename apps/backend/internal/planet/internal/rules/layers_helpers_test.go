package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func setNamed(t *testing.T, name rules.SetName) rules.Set {
	t.Helper()

	set, err := rules.Snapshot{
		Name:  name,
		Tempo: 3,
		Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
	}.Restore()
	require.NoError(t, err)
	return set
}
