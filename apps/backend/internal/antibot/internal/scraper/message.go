package scraper

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "scraper", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetScraper()) }}

func RulesOf(m *antibotv1.ScraperRules) Rules {
	return Rules{
		Enabled:       m.GetEnabled(),
		MinMaps:       m.GetMinMaps(),
		CertainMaps:   m.GetCertainMaps(),
		CertainOffMap: int(m.GetCertainOffMap()),
	}
}
