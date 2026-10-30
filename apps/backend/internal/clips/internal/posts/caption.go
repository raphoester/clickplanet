package posts

import (
	"slices"
	"strings"
)

type CallToAction struct {
	// Kept with the post, so the calls can be compared.
	ID    string
	Words string
}

var callsToAction = []CallToAction{
	{ID: "join-the-battle", Words: "Join the battle now"},
	{ID: "browser", Words: "Play now in your browser"},
	{ID: "no-install", Words: "Play now (no install needed)"},
	{ID: "free", Words: "Play for free"},
	{ID: "pick-a-side", Words: "Pick a side and play now"},
	{ID: "your-flag", Words: "Fight for your flag now"},
}

var gameTags = []string{
	"clickplanet", "rplace", "wplace", "pixelwar", "map", "geographygame", "conquestgame", "videogame", "clickergame", "viral",
}

const (
	site = "https://clickplanet.lol"
	// ?s=yt is how the game counts the players that came from YouTube.
	youtubeSite = site + "/?s=yt"
	brand       = " | CLICKPLANET"
)

type Dice interface {
	// From 0 to n-1.
	IntN(n int) int
}

type wording struct {
	teaser string
	// The name of its wording.
	teaserName string
	call       CallToAction
}

func wordingOf(clip Clip, dice Dice) wording {
	teaser := dice.IntN(len(clip.teasers))
	name := unnamedTeaser
	if len(clip.teaserNames) == len(clip.teasers) {
		name = clip.teaserNames[teaser]
	}
	return wording{teaser: clip.teasers[teaser], teaserName: name, call: callsToAction[dice.IntN(len(callsToAction))]}
}

type TikTokWords struct {
	Caption string
}

// A teaser that says where it happens, the players it names, a call with the site, the music's credit, then the tags.
func (w wording) tiktok(clip Clip) TikTokWords {
	lines := append(append([]string{w.teaser + brand}, fightersLine(clip)...), w.callAndCredit(clip, site)...)
	tags := append(tagsOf(clip), communityTagsOf(clip)...)
	return TikTokWords{Caption: strings.Join(append(lines, hashtagged(tags)), "\n")}
}

// A player searching their name finds the clip; none when it names nobody.
func fightersLine(clip Clip) []string {
	if len(clip.fighters) == 0 {
		return nil
	}
	named := make([]string, 0, len(clip.fighters))
	for _, fighter := range clip.fighters {
		named = append(named, strings.TrimSpace(fighter.Name+" "+flagOf(fighter.Flag)))
	}
	return []string{"⚔️ Top fighters: " + strings.Join(named, ", ")}
}

func (w wording) callAndCredit(clip Clip, link string) []string {
	lines := []string{w.call.Words + " 👉 " + link}
	if clip.attribution != "" {
		lines = append(lines, "🎵 "+clip.attribution)
	}
	return lines
}

func tagsOf(clip Clip) []string {
	tags := slices.Clone(gameTags)
	if clip.continentTag != "" {
		tags = append(tags, clip.continentTag)
	}
	for _, country := range clip.countries {
		tags = append(tags, country.Tag)
	}
	return tags
}

func communityTagsOf(clip Clip) []string {
	var tags []string
	for _, country := range clip.countries {
		if country.Code == clip.key.Flag {
			tags = append(tags, country.Tag+"tiktok"+flagOf(country.Code))
		}
	}
	return tags
}

func hashtagged(tags []string) string {
	marked := make([]string, 0, len(tags))
	for _, tag := range tags {
		marked = append(marked, "#"+tag)
	}
	return strings.Join(marked, " ")
}

// ISO's x codes are for private use: they have no flag.
func flagOf(code string) string {
	if len(code) != 2 || code[0] == 'x' || code[0] < 'a' || code[0] > 'z' || code[1] < 'a' || code[1] > 'z' {
		return ""
	}
	const regionalA = 0x1F1E6
	return string([]rune{regionalA + rune(code[0]-'a'), regionalA + rune(code[1]-'a')})
}
