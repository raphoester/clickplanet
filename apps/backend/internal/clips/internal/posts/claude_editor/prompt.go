package claude_editor

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

// The user's turn: everything after the cached instructions, which changes every run.
type prompt struct {
	Now         time.Time         `json:"now"`
	Slots       int               `json:"slots"`
	ToldFirst   []string          `json:"toldFirst,omitempty"`
	Window      window            `json:"window"`
	Names       map[string]string `json:"names"`
	Continents  []string          `json:"continents"`
	Anthems     []string          `json:"anthems"`
	Fronts      []front           `json:"fronts"`
	Tours       []tour            `json:"tours"`
	Places      []place           `json:"places"`
	RecentPosts []recent          `json:"recentPosts"`
	ShownLately map[string]int    `json:"shownLately"`
}

type window struct {
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

type front struct {
	Front        int       `json:"front"`
	Scope        string    `json:"scope,omitempty"`
	Since        time.Time `json:"since"`
	Until        time.Time `json:"until"`
	Hours        float64   `json:"hours"`
	TilesChanged int       `json:"tilesChanged"`
	Captured     int       `json:"captured"`
	Countries    []country `json:"countries"`
	Flags        []flag    `json:"flags"`
	Bombs        int       `json:"bombs"`
	Spreads      int       `json:"spreads"`
	Enclosures   int       `json:"enclosures"`
	Score        float64   `json:"score"`
}

type country struct {
	Code         string         `json:"code"`
	Continent    string         `json:"continent"`
	Tiles        int            `json:"tiles"`
	Changed      int            `json:"changed"`
	HoldersStart map[string]int `json:"holdersStart"`
	HoldersEnd   map[string]int `json:"holdersEnd"`
	Takers       map[string]int `json:"takers"`
}

type flag struct {
	Code               string         `json:"code"`
	Continent          string         `json:"continent"`
	Took               int            `json:"took"`
	Filled             int            `json:"filled"`
	Lost               int            `json:"lost"`
	TookFrom           map[string]int `json:"tookFrom"`
	Solidity           float64        `json:"solidity"`
	FlippedByContinent map[string]int `json:"flippedByContinent"`
	Fighters           []fighter      `json:"fighters,omitempty"`
}

type fighter struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
	Took   int    `json:"took"`
}

type tour struct {
	Tour               int            `json:"tour"`
	Flag               string         `json:"flag"`
	Since              time.Time      `json:"since"`
	Until              time.Time      `json:"until"`
	Hours              float64        `json:"hours"`
	Took               int            `json:"took"`
	ByContinent        map[string]int `json:"byContinent"`
	TookFrom           map[string]int `json:"tookFrom"`
	NextTook           int            `json:"nextTook"`
	FlippedByContinent map[string]int `json:"flippedByContinent"`
	Solidity           float64        `json:"solidity"`
	Bombs              int            `json:"bombs"`
	Fighters           []fighter      `json:"fighters,omitempty"`
}

type place struct {
	Place              int            `json:"place"`
	Kind               string         `json:"kind"`
	Code               string         `json:"code,omitempty"`
	Continent          string         `json:"continent,omitempty"`
	Tiles              int            `json:"tiles"`
	Holder             string         `json:"holder"`
	Held               int            `json:"held"`
	Share              float64        `json:"share"`
	Rivals             map[string]int `json:"rivals"`
	Changed            int            `json:"changed"`
	FlippedByContinent map[string]int `json:"flippedByContinent"`
	Fighters           []fighter      `json:"fighters,omitempty"`
}

type recent struct {
	At        time.Time `json:"at"`
	Headline  string    `json:"headline"`
	Key       key       `json:"key"`
	Countries []string  `json:"countries,omitempty"`
}

type key struct {
	Flag      string `json:"flag"`
	Continent string `json:"continent"`
	Angle     string `json:"angle,omitempty"`
	Round     string `json:"round,omitempty"`
}

// JSON with its keys in a fixed order, so the same offer is the same prompt.
func PromptOf(offer posts.Offer) (string, error) {
	findings := offer.Findings
	written := prompt{
		Now: offer.Now.UTC().Truncate(time.Second), Slots: offer.Slots(), ShownLately: offer.Shown,
		Window: window{Since: findings.Since.UTC(), Until: findings.Until.UTC()}, Names: findings.Names,
		Continents:  slices.Sorted(maps.Keys(findings.Continents)),
		Anthems:     findings.Anthems,
		Fronts:      make([]front, 0, len(findings.Fronts)),
		Tours:       make([]tour, 0, len(findings.Tours)),
		Places:      make([]place, 0, len(findings.Places)),
		RecentPosts: make([]recent, 0, len(offer.Recent)),
	}
	for _, number := range offer.First {
		written.ToldFirst = append(written.ToldFirst, findings.Seasonal[number-1].Title)
	}
	for i, found := range findings.Fronts {
		written.Fronts = append(written.Fronts, frontOf(i+1, found))
	}
	for i, found := range findings.Tours {
		written.Tours = append(written.Tours, tour{
			Tour: i + 1, Flag: found.Flag, Since: found.Since.UTC(), Until: found.Until.UTC(), Hours: hoursOf(found.Since, found.Until),
			Took: found.Took, ByContinent: found.ByContinent, TookFrom: found.TookFrom, NextTook: found.NextTook,
			FlippedByContinent: found.FlippedByContinent, Solidity: found.Solidity, Bombs: found.Bombs,
			Fighters: fightersOf(found.Fighters),
		})
	}
	for i, found := range findings.Places {
		written.Places = append(written.Places, place{
			Place: i + 1, Kind: string(found.Kind), Code: found.Code, Continent: found.Continent, Tiles: found.Tiles,
			Holder: found.Holder, Held: found.Held, Share: found.Share, Rivals: found.Rivals, Changed: found.Changed,
			FlippedByContinent: found.FlippedByContinent,
			Fighters:           fightersOf(found.Fighters),
		})
	}
	for _, telling := range offer.Recent {
		written.RecentPosts = append(written.RecentPosts, recent{
			At: telling.At.UTC(), Headline: telling.Title, Key: keyOf(telling.Key), Countries: telling.Countries,
		})
	}

	text, err := json.Marshal(written)
	if err != nil {
		return "", fmt.Errorf("failed to write the prompt: %w", err)
	}
	return "The run's findings, as JSON:\n\n" + string(text), nil
}

func frontOf(number int, found posts.Front) front {
	written := front{
		Front: number, Scope: found.Scope, Since: found.Since.UTC(), Until: found.Until.UTC(), Hours: hoursOf(found.Since, found.Until),
		TilesChanged: found.TilesChanged, Captured: found.Captured, Countries: make([]country, 0, len(found.Countries)),
		Flags: make([]flag, 0, len(found.Flags)), Bombs: found.Bombs, Spreads: found.Spreads, Enclosures: found.Enclosures,
		Score: found.Score,
	}
	for _, c := range found.Countries {
		written.Countries = append(written.Countries, country{
			Code: c.Code, Continent: c.Continent, Tiles: c.Tiles, Changed: c.Changed,
			HoldersStart: c.HoldersStart, HoldersEnd: c.HoldersEnd, Takers: c.Takers,
		})
	}
	for _, f := range found.Flags {
		written.Flags = append(written.Flags, flag{
			Code: f.Code, Continent: f.Continent, Took: f.Took, Filled: f.Filled, Lost: f.Lost, TookFrom: f.TookFrom, Solidity: f.Solidity,
			FlippedByContinent: f.FlippedByContinent,
			Fighters:           fightersOf(f.Fighters),
		})
	}
	return written
}

func fightersOf(found []posts.Fighter) []fighter {
	written := make([]fighter, 0, len(found))
	for _, f := range found {
		written = append(written, fighter{Number: int(f.Number), Name: f.Name, Took: f.Took})
	}
	return written
}

func hoursOf(since, until time.Time) float64 {
	return float64(until.Sub(since).Round(6*time.Minute)) / float64(time.Hour)
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
