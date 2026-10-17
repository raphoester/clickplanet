package posts

import (
	"strings"
	"unicode/utf8"
)

const (
	youtubeTitleLength = 100
	// Past it, YouTube ignores every hashtag of the video.
	youtubeHashtags = 60
	// With a comma between two tags, as YouTube counts them.
	youtubeTagsLength = 500
)

type YouTubeWords struct {
	Title       string
	Description string
	Tags        []string
}

// No #Shorts: a vertical video of 3 minutes or less is one by itself. No TikTok community tag either.
func (w wording) youtube(clip Clip) YouTubeWords {
	tags := tagsOf(clip)
	lines := append(w.callAndCredit(clip), hashtagged(tags[:min(len(tags), youtubeHashtags)]))
	return YouTubeWords{Title: w.youtubeTitle(), Description: strings.Join(lines, "\n"), Tags: within(tags, youtubeTagsLength)}
}

func (w wording) youtubeTitle() string {
	if title := w.teaser + brand; utf8.RuneCountInString(title) <= youtubeTitleLength {
		return title
	}
	return cut(w.teaser, youtubeTitleLength)
}

func cut(text string, length int) string {
	runes := []rune(text)
	if len(runes) <= length {
		return text
	}
	return string(runes[:length-1]) + "…"
}

func within(tags []string, length int) []string {
	used := 0
	for i, tag := range tags {
		if i > 0 {
			used++
		}
		used += utf8.RuneCountInString(tag)
		if used > length {
			return tags[:i]
		}
	}
	return tags
}
