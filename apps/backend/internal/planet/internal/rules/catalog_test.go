package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

var defaults = rules.Set{
	Name:    rules.Default,
	Tempo:   1,
	Boxes:   bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
	AntiBot: rules.AntiBot{Crowd: 6},
}

var rushConfig = rules.Config{Sets: map[rules.SetName]rules.SetConfig{
	"rush": {
		Tempo:   3,
		Boxes:   rules.BoxesConfig{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
		AntiBot: rules.AntiBotConfig{Crowd: 50},
	},
}}

func TestASetConfiguredInFullIsTakenAsIs(t *testing.T) {
	catalog, err := rules.NewCatalog(rushConfig, defaults)
	require.NoError(t, err)

	rush, err := catalog.Set("rush")
	require.NoError(t, err)
	assert.Equal(t, rules.Set{
		Name:    "rush",
		Tempo:   3,
		Boxes:   bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
		AntiBot: rules.AntiBot{Crowd: 50},
	}, rush)
}

func TestWhatASetLeavesOutIsTheDefaults(t *testing.T) {
	catalog, err := rules.NewCatalog(rules.Config{Sets: map[rules.SetName]rules.SetConfig{
		"quick": {Tempo: 2},
		"boxes": {Boxes: rules.BoxesConfig{MaxChargesPerHour: 20}},
	}}, defaults)
	require.NoError(t, err)

	quick, err := catalog.Set("quick")
	require.NoError(t, err)
	assert.Equal(t, rules.Set{Name: "quick", Tempo: 2, Boxes: defaults.Boxes, AntiBot: defaults.AntiBot}, quick)

	boxes, err := catalog.Set("boxes")
	require.NoError(t, err)
	assert.InDelta(t, 1.0, boxes.Tempo, 1e-9)
	assert.Equal(t, 20, boxes.Boxes.MaxChargesPerHour)
	assert.Equal(t, 4*time.Minute, boxes.Boxes.MinInterval)
}

func TestASetFreezesTheMapOnlyWhenItSaysSo(t *testing.T) {
	catalog, err := rules.NewCatalog(rules.Config{Sets: map[rules.SetName]rules.SetConfig{
		"frozen": {Frozen: true},
	}}, defaults)
	require.NoError(t, err)

	frozen, err := catalog.Set("frozen")
	require.NoError(t, err)
	assert.Equal(t, rules.Set{Name: "frozen", Tempo: 1, Boxes: defaults.Boxes, AntiBot: defaults.AntiBot, Frozen: true}, frozen)

	open, err := catalog.Set(rules.Default)
	require.NoError(t, err)
	assert.False(t, open.Frozen)
}

func TestTheDefaultSetIsTheConfigsOwnRules(t *testing.T) {
	catalog, err := rules.NewCatalog(rushConfig, defaults)
	require.NoError(t, err)

	set, err := catalog.Set(rules.Default)
	require.NoError(t, err)
	assert.Equal(t, defaults, set)
}

func TestAnUnknownSetIsRefused(t *testing.T) {
	catalog, err := rules.NewCatalog(rushConfig, defaults)
	require.NoError(t, err)

	_, err = catalog.Set("frozen")
	require.ErrorIs(t, err, rules.ErrUnknownSet)
}

func TestASetWhoseShortestWaitPassesItsLongestIsRefused(t *testing.T) {
	_, err := rules.NewCatalog(rules.Config{Sets: map[rules.SetName]rules.SetConfig{
		"odd": {Boxes: rules.BoxesConfig{MinInterval: 10 * time.Minute}},
	}}, defaults)

	require.ErrorContains(t, err, "rules.sets.odd")
}

func TestTheSetsAreListedDefaultFirstThenByName(t *testing.T) {
	catalog, err := rules.NewCatalog(rules.Config{Sets: map[rules.SetName]rules.SetConfig{
		"rush": {Tempo: 3}, "calm": {Tempo: 1}, "zen": {Tempo: 1},
	}}, defaults)
	require.NoError(t, err)

	sets := catalog.Sets()
	names := make([]rules.SetName, 0, len(sets))
	for _, set := range sets {
		names = append(names, set.Name)
	}
	assert.Equal(t, []rules.SetName{rules.Default, "calm", "rush", "zen"}, names)
}

func TestTheSetOfAChangeIsTheSetItNamedAndNoChangeIsTheDefault(t *testing.T) {
	catalog, err := rules.NewCatalog(rushConfig, defaults)
	require.NoError(t, err)

	none, err := catalog.SetOf(rules.Change{})
	require.NoError(t, err)
	assert.Equal(t, defaults, none)

	rush, err := catalog.SetOf(rules.Snapshot{Request: "r", Set: "rush", Source: rules.SourceInternal}.Restore())
	require.NoError(t, err)
	assert.Equal(t, rules.SetName("rush"), rush.Name)

	_, err = catalog.SetOf(rules.Snapshot{Request: "r", Set: "gone", Source: rules.SourceInternal}.Restore())
	require.ErrorIs(t, err, rules.ErrUnknownSet)
}

func TestTheConfigRefusesWhatCannotBeASet(t *testing.T) {
	for name, config := range map[string]rules.Config{
		"default redefined": {Sets: map[rules.SetName]rules.SetConfig{rules.Default: {Tempo: 2}}},
		"bad name":          {Sets: map[rules.SetName]rules.SetConfig{"Rush!": {Tempo: 2}}},
		"negative tempo":    {Sets: map[rules.SetName]rules.SetConfig{"slow": {Tempo: -1}}},
		"negative boxes":    {Sets: map[rules.SetName]rules.SetConfig{"slow": {Boxes: rules.BoxesConfig{MaxChargesPerHour: -1}}}},
		"negative crowd":    {Sets: map[rules.SetName]rules.SetConfig{"slow": {AntiBot: rules.AntiBotConfig{Crowd: -1}}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, config.Validate())
		})
	}

	require.NoError(t, rushConfig.Validate())
	require.NoError(t, rules.Config{}.Validate())
}
