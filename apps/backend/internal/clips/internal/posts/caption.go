package posts

import (
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

const site = "clickplanet.lol"

type Dice interface {
	// From 0 to n-1.
	IntN(n int) int
}

type wording struct {
	teaser string
	call   CallToAction
}

func wordingOf(clip Clip, dice Dice) wording {
	return wording{teaser: clip.teasers[dice.IntN(len(clip.teasers))], call: callsToAction[dice.IntN(len(callsToAction))]}
}

// A teaser that says where it happens, a call with the site, the music's credit, then the tags.
func (w wording) caption(clip Clip) string {
	lines := []string{w.teaser + " | CLICKPLANET", w.call.Words + " 👉 " + site}
	if clip.attribution != "" {
		lines = append(lines, "🎵 "+clip.attribution)
	}
	return strings.Join(append(lines, strings.Join(hashtagsOf(clip), " ")), "\n")
}

func hashtagsOf(clip Clip) []string {
	tags := make([]string, 0, len(gameTags)+len(clip.countries)+2)
	for _, tag := range gameTags {
		tags = append(tags, "#"+tag)
	}
	if clip.continentTag != "" {
		tags = append(tags, "#"+clip.continentTag)
	}
	for _, country := range clip.countries {
		tags = append(tags, "#"+country.Tag)
	}
	for _, country := range clip.countries {
		if country.Code == clip.key.Flag {
			tags = append(tags, "#"+country.Tag+"tiktok"+flagOf(country.Code))
		}
	}
	return tags
}

// ISO's x codes are for private use: they have no flag.
func flagOf(code string) string {
	if len(code) != 2 || code[0] == 'x' || code[0] < 'a' || code[0] > 'z' || code[1] < 'a' || code[1] > 'z' {
		return ""
	}
	const regionalA = 0x1F1E6
	return string([]rune{regionalA + rune(code[0]-'a'), regionalA + rune(code[1]-'a')})
}
