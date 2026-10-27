//go:build testing

package rules

import (
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
)

// Seeded is the set the migration keeps as default: no round of its own.
func Seeded() Snapshot {
	return Snapshot{Name: Default, Rules: &scoringv1.Rules{OwnRound: &scoringv1.OwnRoundRules{Enabled: false, Times: 1}}}
}

// Battle is a set that makes the window of the entry that plays it a round of its own, worth times a day's points.
func Battle(name SetName, times uint32) Snapshot {
	return Snapshot{Name: name, Rules: &scoringv1.Rules{OwnRound: &scoringv1.OwnRoundRules{Enabled: true, Times: times}}}
}
