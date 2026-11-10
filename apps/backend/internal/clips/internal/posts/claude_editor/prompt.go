package claude_editor

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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
	Sagas       []saga            `json:"sagas"`
	Paths       paths             `json:"paths"`
	Told        []told            `json:"told"`
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

type saga struct {
	Saga       int         `json:"saga"`
	Kind       string      `json:"kind"`
	Code       string      `json:"code,omitempty"`
	Continent  string      `json:"continent,omitempty"`
	Tiles      int         `json:"tiles"`
	Holder     string      `json:"holder"`
	Share      float64     `json:"share"`
	Since      time.Time   `json:"since"`
	Until      time.Time   `json:"until"`
	Reigns     []reign     `json:"reigns"`
	Contenders []contender `json:"contenders"`
}

type contender struct {
	Flag  string     `json:"flag"`
	Since time.Time  `json:"since"`
	Until *time.Time `json:"until,omitempty"`
	Peak  float64    `json:"peak"`
	Share float64    `json:"share"`
}

// Each flag's land at each of at, in percent of each continent's tiles and of the world's: a continent it never held 1%
// of is left out.
type paths struct {
	At    []time.Time `json:"at"`
	Flags []path      `json:"flags"`
}

type path struct {
	Flag      string           `json:"flag"`
	PercentOf map[string][]int `json:"percentOf"`
}

// days: how long it held over half, to its until or to the saga's. conquestHours: from its push to holding over half.
type reign struct {
	Reign         int        `json:"reign"`
	Flag          string     `json:"flag"`
	PushedAt      *time.Time `json:"pushedAt,omitempty"`
	Since         time.Time  `json:"since"`
	Until         *time.Time `json:"until,omitempty"`
	Days          float64    `json:"days"`
	ConquestHours float64    `json:"conquestHours,omitempty"`
	Peak          float64    `json:"peak"`
	Share         float64    `json:"share"`
	TookFrom      string     `json:"tookFrom,omitempty"`
	TookFromShare float64    `json:"tookFromShare,omitempty"`
	LostTo        string     `json:"lostTo,omitempty"`
}

// A post, and what its clip told when its record kept it: else only the countries it showed (shown).
type told struct {
	At           time.Time  `json:"at"`
	Headline     string     `json:"headline"`
	Line         string     `json:"line,omitempty"`
	Key          key        `json:"key"`
	Kind         string     `json:"kind,omitempty"`
	Attacker     string     `json:"attacker,omitempty"`
	Rival        string     `json:"rival,omitempty"`
	Countries    []string   `json:"countries,omitempty"`
	Continent    string     `json:"continent,omitempty"`
	Since        *time.Time `json:"since,omitempty"`
	Until        *time.Time `json:"until,omitempty"`
	ShareAtStart *float64   `json:"shareAtStart,omitempty"`
	ShareAtEnd   *float64   `json:"shareAtEnd,omitempty"`
	Shown        []string   `json:"shown,omitempty"`
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
		Continents: slices.Sorted(maps.Keys(findings.Continents)),
		Anthems:    findings.Anthems,
		Fronts:     make([]front, 0, len(findings.Fronts)),
		Tours:      make([]tour, 0, len(findings.Tours)),
		Places:     make([]place, 0, len(findings.Places)),
		Sagas:      make([]saga, 0, len(findings.Sagas)),
		Paths:      pathsOf(findings),
		Told:       make([]told, 0, len(offer.Told)),
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
	for i, found := range findings.Sagas {
		written.Sagas = append(written.Sagas, sagaOf(i+1, found))
	}
	for _, telling := range offer.Told {
		written.Told = append(written.Told, toldOf(telling))
	}

	text, err := json.Marshal(written)
	if err != nil {
		return "", fmt.Errorf("failed to write the prompt: %w", err)
	}
	return "The run's findings, as JSON:\n\n" + string(text), nil
}

func toldOf(telling posts.Telling) told {
	written := told{At: telling.At.UTC(), Headline: telling.Title, Key: keyOf(telling.Key)}
	facts := telling.Facts
	if facts.Attacker == "" {
		written.Shown = telling.Countries
		return written
	}
	written.Line, written.Kind, written.Attacker, written.Rival = facts.Line, facts.Kind, facts.Attacker, facts.Rival
	written.Countries, written.Continent = facts.Countries, facts.Region
	written.Since, written.Until = setOf(facts.Since), setOf(facts.Until)
	start, end := math.Round(facts.ShareAtStart*1000)/1000, math.Round(facts.ShareAtEnd*1000)/1000
	written.ShareAtStart, written.ShareAtEnd = &start, &end
	return written
}

// Every path's marks at one list of times: a flag with no mark at a time held nothing then.
func pathsOf(findings posts.Findings) paths {
	times := cpcolls.NewSet[time.Time]()
	for _, found := range findings.Paths {
		for _, mark := range found.Marks {
			times.Add(mark.At.UTC())
		}
	}
	at := make([]time.Time, 0, times.Len())
	times.ForEach(func(when time.Time) { at = append(at, when) })
	slices.SortFunc(at, time.Time.Compare)
	index := make(map[time.Time]int, len(at))
	for i, when := range at {
		index[when] = i
	}
	world := 0
	for _, tiles := range findings.ContinentTiles {
		world += tiles
	}

	written := paths{At: at, Flags: make([]path, 0, len(findings.Paths))}
	for _, found := range findings.Paths {
		held := map[string][]int{}
		percent := func(continent string, i, tiles, of int) {
			if of == 0 {
				return
			}
			if held[continent] == nil {
				held[continent] = make([]int, len(at))
			}
			held[continent][i] = int(math.Round(float64(tiles) * 100 / float64(of)))
		}
		for _, mark := range found.Marks {
			i, total := index[mark.At.UTC()], 0
			for continent, tiles := range mark.TilesByContinent {
				percent(continent, i, tiles, findings.ContinentTiles[continent])
				total += tiles
			}
			percent(posts.TheWorld, i, total, world)
		}
		maps.DeleteFunc(held, func(_ string, percents []int) bool { return slices.Max(percents) == 0 })
		written.Flags = append(written.Flags, path{Flag: found.Flag, PercentOf: held})
	}
	return written
}

func sagaOf(number int, found posts.Saga) saga {
	written := saga{
		Saga: number, Kind: string(found.Kind), Code: found.Code, Continent: found.Continent, Tiles: found.Tiles, Holder: found.Holder,
		Share: found.Share, Since: found.Since.UTC(), Until: found.Until.UTC(), Reigns: make([]reign, 0, len(found.Reigns)),
	}
	for i, r := range found.Reigns {
		end := found.Until
		if !r.Until.IsZero() {
			end = r.Until
		}
		told := reign{
			Reign: i + 1, Flag: r.Flag, PushedAt: setOf(r.PushedAt), Since: r.Since.UTC(), Until: setOf(r.Until),
			Days: float64(end.Sub(r.Since).Round(144*time.Minute)) / float64(24*time.Hour), Peak: r.Peak, Share: r.Share,
			TookFrom: r.TookFrom, TookFromShare: r.TookFromShare, LostTo: r.LostTo,
		}
		if !r.PushedAt.IsZero() {
			told.ConquestHours = hoursOf(r.PushedAt, r.Since)
		}
		written.Reigns = append(written.Reigns, told)
	}
	written.Contenders = make([]contender, 0, len(found.Contenders))
	for _, c := range found.Contenders {
		written.Contenders = append(written.Contenders, contender{Flag: c.Flag, Since: c.Since.UTC(), Until: setOf(c.Until), Peak: c.Peak, Share: c.Share})
	}
	return written
}

// Unset is left out.
func setOf(at time.Time) *time.Time {
	if at.IsZero() {
		return nil
	}
	utc := at.UTC()
	return &utc
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
