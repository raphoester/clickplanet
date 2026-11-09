package teasers_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

func TestATeaserSaysWhenItsBattleStartsAndItsRulesOnEachPlatform(t *testing.T) {
	words := ringWar().Words()

	assert.Equal(t, teasers.Words{
		Title: "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC",
		TikTok: teasers.TikTokWords{
			Title: "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC",
			Caption: "Battle on Saturday 14 November at 20:00 UTC ⚔️\n" +
				"Close a ring around any land.\n" +
				"Swipe for the rules, then join the battle 👉 https://clickplanet.lol\n" +
				"#clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #battle",
		},
		YouTube: teasers.YouTubeWords{
			Title: "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC | CLICKPLANET",
			Description: "Battle on Saturday 14 November at 20:00 UTC ⚔️\n" +
				"Close a ring around any land.\n" +
				"• Circle any land with your flag: Every click can close a ring.\n" +
				"• Gain the most land\n" +
				"Join the battle 👉 https://clickplanet.lol/?s=yt\n" +
				"#clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #battle",
			Tags: []string{
				"clickplanet", "rplace", "wplace", "pixelwar", "map", "geographygame", "conquestgame", "videogame",
				"clickergame", "battle",
			},
		},
	}, words)
}

func TestTheFinalBattleIsNamedSoAndTaggedSo(t *testing.T) {
	finale := ringWar()
	finale.EndsSeason, finale.Description = true, ""

	words := finale.Words()

	assert.Equal(t, "FINAL BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC", words.TikTok.Title)
	assert.True(t, strings.HasPrefix(words.TikTok.Caption, "Final Battle on Saturday 14 November at 20:00 UTC ⚔️\nSwipe"))
	assert.Equal(t, "finalbattle", words.YouTube.Tags[len(words.YouTube.Tags)-1])
	assert.NotContains(t, ringWar().Words().YouTube.Tags, "finalbattle", "the finale's tag is its own")
}

func TestATitleIsCutToWhatEachPlatformTakes(t *testing.T) {
	words := ringWar().Words()

	assert.LessOrEqual(t, utf8.RuneCountInString(words.TikTok.Title), 90)
	assert.LessOrEqual(t, utf8.RuneCountInString(words.YouTube.Title), 100)
}
