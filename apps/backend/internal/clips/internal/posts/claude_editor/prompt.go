package claude_editor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

// What the run offers, as the user's turn: everything after the cached instructions, which changes every run.
type prompt struct {
	Now         time.Time      `json:"now"`
	Slots       int            `json:"slots"`
	ToldFirst   []string       `json:"toldFirst,omitempty"`
	Stories     []offered      `json:"stories"`
	RecentPosts []recent       `json:"recentPosts"`
	ShownLately map[string]int `json:"shownLately"`
}

type offered struct {
	Pick      posts.Pick      `json:"pick"`
	Headline  string          `json:"headline"`
	Key       key             `json:"key"`
	Countries []string        `json:"countries"`
	Teasers   []string        `json:"teasers,omitempty"`
	Facts     json.RawMessage `json:"facts,omitempty"`
}

type recent struct {
	At       time.Time `json:"at"`
	Headline string    `json:"headline"`
	Key      key       `json:"key"`
}

type key struct {
	Flag      string `json:"flag"`
	Continent string `json:"continent"`
	Angle     string `json:"angle,omitempty"`
	Round     string `json:"round,omitempty"`
}

// The run's offer as the model reads it: JSON, its keys in a fixed order, so the same offer is the same prompt.
func PromptOf(offer posts.Offer) (string, error) {
	written := prompt{
		Now: offer.Now.UTC().Truncate(time.Second), Slots: offer.Slots(), ShownLately: offer.Shown,
		Stories: make([]offered, 0, len(offer.Stories)), RecentPosts: make([]recent, 0, len(offer.Recent)),
	}
	for _, story := range offer.First {
		written.ToldFirst = append(written.ToldFirst, story.Title)
	}
	for _, story := range offer.Stories {
		written.Stories = append(written.Stories, offered{
			Pick: story.Pick, Headline: story.Title, Key: keyOf(story.Key), Countries: story.Countries,
			Teasers: story.Teasers, Facts: json.RawMessage(story.Facts),
		})
	}
	for _, telling := range offer.Recent {
		written.RecentPosts = append(written.RecentPosts, recent{At: telling.At.UTC(), Headline: telling.Title, Key: keyOf(telling.Key)})
	}

	text, err := json.Marshal(written)
	if err != nil {
		return "", fmt.Errorf("failed to write the prompt: %w", err)
	}
	return "The run's stories, as JSON:\n\n" + string(text), nil
}

func keyOf(written posts.StoryKey) key {
	return key{Flag: written.Flag, Continent: written.Continent, Angle: string(written.Angle), Round: written.Round}
}

// Dollars a million tokens. A prompt over longPrompt tokens, cache included, is billed longRate times both.
const (
	inputRate   = 0.10
	outputRate  = 0.50
	longPrompt  = 100_000
	longRate    = 5
	cacheWrites = 1.25
	cacheReads  = 0.1
)

func spendOf(model string, usage anthropic.Usage) posts.Spend {
	return SpendOf(model, usage.InputTokens, usage.CacheCreationInputTokens, usage.CacheReadInputTokens, usage.OutputTokens)
}

// The cost of a call to Claude Haiku 5.5, from its usage. Another model is priced as it, which only a test would ask.
func SpendOf(model string, input, cacheWrite, cacheRead, output int64) posts.Spend {
	in, out := inputRate, outputRate
	if input+cacheWrite+cacheRead > longPrompt {
		in, out = in*longRate, out*longRate
	}
	usd := (float64(input)*in + float64(cacheWrite)*in*cacheWrites + float64(cacheRead)*in*cacheReads +
		float64(output)*out) / 1_000_000
	return posts.Spend{
		Model: model, InputTokens: input, CacheWriteTokens: cacheWrite, CacheReadTokens: cacheRead, OutputTokens: output, USD: usd,
	}
}
