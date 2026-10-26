package bans

import (
	"time"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "bans", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetBans()) }}

func RulesOf(m *antibotv1.BanRules) Rules {
	durations := make([]time.Duration, 0, len(m.GetBanDurations()))
	for _, d := range m.GetBanDurations() {
		durations = append(durations, d.AsDuration())
	}

	return Rules{
		Enforce:        m.GetEnforce(),
		BanDurations:   durations,
		ReflagInterval: m.GetReflagInterval().AsDuration(),
	}
}
