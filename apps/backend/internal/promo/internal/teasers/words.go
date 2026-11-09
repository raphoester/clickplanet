package teasers

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	site = "https://clickplanet.lol"
	// ?s=yt is how the game counts the players that came from YouTube.
	youtubeSite = site + "/?s=yt"
	brand       = " | CLICKPLANET"

	// The title over a TikTok carousel: TikTok strips its hashtags and links.
	tiktokTitleLength  = 90
	youtubeTitleLength = 100
)

// The clips' tags, and the battle's own.
var tags = []string{
	"clickplanet", "rplace", "wplace", "pixelwar", "map", "geographygame", "conquestgame", "videogame", "clickergame",
	"battle",
}

// What a teaser says on each platform.
type Words struct {
	// The post's, in the service's dashboard and in ListPosts.
	Title   string
	TikTok  TikTokWords
	YouTube YouTubeWords
}

type TikTokWords struct {
	// Over the photos.
	Title string
	// Below them.
	Caption string
}

type YouTubeWords struct {
	Title       string
	Description string
	Tags        []string
}

// When the battle starts, as its frame on the cards says it: in UTC, for players all over the world.
func (t Teaser) Words() Words {
	when := t.name() + " on " + dayOf(t.StartsAt) + " at " + timeOf(t.StartsAt)
	headline := strings.ToUpper(when)
	marked := hashtagged(t.tags())

	tiktok := []string{when + " ⚔️"}
	youtube := []string{when + " ⚔️"}
	if t.Description != "" {
		tiktok = append(tiktok, t.Description)
		youtube = append(youtube, t.Description)
	}
	tiktok = append(tiktok, "Swipe for the rules, then join the battle 👉 "+site, marked)
	for _, card := range t.Cards {
		rule := "• " + card.Headline
		if card.Line != "" {
			rule += ": " + card.Line
		}
		youtube = append(youtube, rule)
	}
	youtube = append(youtube, "Join the battle 👉 "+youtubeSite, marked)

	return Words{
		Title:  headline,
		TikTok: TikTokWords{Title: cut(headline, tiktokTitleLength), Caption: strings.Join(tiktok, "\n")},
		YouTube: YouTubeWords{
			Title: cut(headline+brand, youtubeTitleLength), Description: strings.Join(youtube, "\n"), Tags: t.tags(),
		},
	}
}

func (t Teaser) name() string {
	if t.EndsSeason {
		return "Final Battle"
	}
	return "Battle"
}

func (t Teaser) tags() []string {
	if t.EndsSeason {
		return append(append([]string(nil), tags...), "finalbattle")
	}
	return tags
}

// Saturday 14 November.
func dayOf(at time.Time) string {
	return at.UTC().Format("Monday 2 January")
}

// 20:00 UTC.
func timeOf(at time.Time) string {
	return at.UTC().Format("15:04") + " UTC"
}

func hashtagged(tags []string) string {
	marked := make([]string, 0, len(tags))
	for _, tag := range tags {
		marked = append(marked, "#"+tag)
	}
	return strings.Join(marked, " ")
}

func cut(text string, length int) string {
	runes := []rune(text)
	if utf8.RuneCountInString(text) <= length {
		return text
	}
	return string(runes[:length-1]) + "…"
}
