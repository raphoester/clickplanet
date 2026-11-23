// Package claude_editor asks a Claude model to make a run's stories from the findings, and to headline them.
package claude_editor

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

// What makes a story worth telling and a headline good: the prose the owner tunes, sent first and cached.
//
//go:embed instructions.md
var Instructions string

const (
	// The cheapest model, at $0.10 and $0.50 a million tokens in and out while a prompt stays at 100K or fewer.
	DefaultModel = "claude-haiku-5-5"

	// Its thinking counts: with the lab's story a run used 8,000 and more; 16,000 is about 80 s, inside the timeout.
	maxTokens = 16000
	// One call a run, in a job of 10 minutes: a model that does not answer in 2 leaves the run quiet.
	timeout = 2 * time.Minute
	retries = 1
)

type Editor struct {
	client *anthropic.Client
	model  string
}

var _ choose_clips_usecase.Editor = (*Editor)(nil)

func New(client *anthropic.Client, model string) *Editor {
	return &Editor{client: client, model: model}
}

var (
	ErrRefused  = errors.New("the model refused")
	ErrNoAnswer = errors.New("the model gave no answer the run can read")
)

var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

func (e *Editor) Edit(ctx context.Context, offer posts.Offer) (posts.Edition, posts.Spend, error) {
	prompt, err := PromptOf(offer)
	if err != nil {
		return posts.Edition{}, posts.Spend{}, err
	}

	message, err := e.client.Messages.New(ctx, e.paramsOf(prompt, formatOf(offer)), requestOptions...)
	if err != nil {
		// Sent, and billed as nothing that can be known here.
		return posts.Edition{}, posts.Spend{Model: e.model}, fmt.Errorf("the model could not be asked: %w", cpanthropic.Redacted(err))
	}
	spend := spendOf(e.model, message.Usage)

	switch message.StopReason {
	case anthropic.StopReasonRefusal:
		// Haiku has no fallback of its own on the server: a refusal is the end of it.
		return posts.Edition{}, spend, fmt.Errorf("%w (%s)", ErrRefused, cmpOr(string(message.StopDetails.Category), "no category"))
	case anthropic.StopReasonEndTurn:
	default:
		return posts.Edition{}, spend, fmt.Errorf("%w: it stopped on %s", ErrNoAnswer, message.StopReason)
	}

	edition, err := editionOf(message.Content)
	return edition, spend, err
}

// The lab's, which shapes its answer; a run with no lab answers its stories as a clip's.
func formatOf(offer posts.Offer) posts.Format {
	if !offer.Lab.On() {
		return posts.GlobeReplay
	}
	return offer.Lab.Experiment.Format
}

func (e *Editor) paramsOf(prompt string, format posts.Format) anthropic.MessageNewParams {
	return anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: Instructions, CacheControl: anthropic.NewCacheControlEphemeralParam()}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchemaOf(format)},
		},
	}
}

// The prompt's size, as the model counts it, with no answer asked for: what `clips edit -prompt` shows.
func (e *Editor) Count(ctx context.Context, offer posts.Offer) (int64, error) {
	prompt, err := PromptOf(offer)
	if err != nil {
		return 0, err
	}
	params := e.paramsOf(prompt, formatOf(offer))
	counted, err := e.client.Messages.CountTokens(ctx, anthropic.MessageCountTokensParams{
		Model:        params.Model,
		System:       anthropic.MessageCountTokensParamsSystemUnion{OfTextBlockArray: params.System},
		Messages:     params.Messages,
		OutputConfig: params.OutputConfig,
	}, requestOptions...)
	if err != nil {
		return 0, fmt.Errorf("the prompt could not be counted: %w", cpanthropic.Redacted(err))
	}
	return counted.InputTokens, nil
}

// What the model answers, as the schema of its structured output holds it to.
type answer struct {
	Why     string          `json:"why"`
	Stories []answeredStory `json:"stories"`
	// Written to the lab's brief: one at most, and none when the run asks for none.
	Lab []answeredStory `json:"lab"`
}

type answeredStory struct {
	Front     int      `json:"front"`
	Tour      int      `json:"tour"`
	Place     int      `json:"place"`
	Saga      int      `json:"saga"`
	Since     string   `json:"since"`
	Kind      string   `json:"kind"`
	Attacker  string   `json:"attacker"`
	Rival     string   `json:"rival"`
	TakenFrom []string `json:"takenFrom"`
	Countries []string `json:"countries"`
	Continent string   `json:"continent"`
	Team      string   `json:"team"`
	Headline  string   `json:"headline"`
	Teasers   []string `json:"teasers"`
	Line      string   `json:"line"`
	Call      string   `json:"call"`
	CallFlags []string `json:"callFlags"`
	Anthem    string   `json:"anthem"`
	Look      string   `json:"look"`
	Fighters  []int    `json:"fighters"`
	// A lab story's alone.
	Hook     string         `json:"hook"`
	Language string         `json:"language"`
	List     string         `json:"list"`
	Ranks    []answeredRank `json:"ranks"`
}

type answeredRank struct {
	Code string `json:"code"`
	Line string `json:"line"`
}

var storyFields = []string{
	"front", "tour", "place", "saga", "since", "kind", "attacker", "rival", "takenFrom", "countries", "continent", "team", "headline", "teasers",
	"line", "call", "callFlags", "anthem", "look", "fighters",
}

// A lab story is a story with its hook and its language.
func storySchema(lab bool) map[string]any {
	required, properties := slices.Clone(storyFields), maps.Clone(storyProperties)
	if lab {
		required = append(required, "hook", "language")
		properties["hook"] = map[string]any{"type": "string"}
		properties["language"] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
}

var carouselSchema = map[string]any{
	"type": "object", "additionalProperties": false, "required": []string{"list", "hook", "ranks"},
	"properties": map[string]any{
		"list": map[string]any{"type": "string", "enum": listNames()},
		"hook": map[string]any{"type": "string"},
		"ranks": map[string]any{"type": "array", "items": map[string]any{
			"type": "object", "additionalProperties": false, "required": []string{"code", "line"},
			"properties": map[string]any{"code": map[string]any{"type": "string"}, "line": map[string]any{"type": "string"}},
		}},
	},
}

func listNames() []string {
	names := make([]string, 0, len(posts.ListNames))
	for _, name := range posts.ListNames {
		names = append(names, string(name))
	}
	return names
}

func answerSchemaOf(format posts.Format) map[string]any {
	lab := storySchema(true)
	if format == posts.Carousel {
		lab = carouselSchema
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"why", "stories", "lab"},
		"properties": map[string]any{
			"why":     map[string]any{"type": "string"},
			"stories": map[string]any{"type": "array", "items": storySchema(false)},
			"lab":     map[string]any{"type": "array", "items": lab},
		},
	}
}

var storyProperties = map[string]any{
	"front":     map[string]any{"type": "integer"},
	"tour":      map[string]any{"type": "integer"},
	"place":     map[string]any{"type": "integer"},
	"saga":      map[string]any{"type": "integer"},
	"since":     map[string]any{"type": "string"},
	"kind":      map[string]any{"type": "string", "enum": []string{"attack", "invasion", "comeback", "kickout", "rout", "battle", ""}},
	"attacker":  map[string]any{"type": "string"},
	"rival":     map[string]any{"type": "string"},
	"takenFrom": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	"countries": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	"continent": map[string]any{"type": "string"},
	"team":      map[string]any{"type": "string"},
	"headline":  map[string]any{"type": "string"},
	"teasers":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	"line":      map[string]any{"type": "string"},
	"call":      map[string]any{"type": "string"},
	"callFlags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	"anthem":    map[string]any{"type": "string"},
	"look":      map[string]any{"type": "string", "enum": []string{"flags", "dive", ""}},
	"fighters":  map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
}

// The answer is the one text block; thinking blocks come before it and are left.
func editionOf(content []anthropic.ContentBlockUnion) (posts.Edition, error) {
	for i := range content {
		text, ok := content[i].AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}
		var answered answer
		if err := json.Unmarshal([]byte(text.Text), &answered); err != nil {
			return posts.Edition{}, fmt.Errorf("%w: %w", ErrNoAnswer, err)
		}
		edition := posts.Edition{Why: strings.TrimSpace(answered.Why)}
		for _, story := range answered.Stories {
			edition.Stories = append(edition.Stories, story.story())
		}
		for _, story := range answered.Lab {
			edition.Lab = append(edition.Lab, story.story())
		}
		return edition, nil
	}
	return posts.Edition{}, fmt.Errorf("%w: no text", ErrNoAnswer)
}

func (s answeredStory) story() posts.Story {
	// None is nil, as the model's [] was before the lab.
	var fighters []posts.FighterNumber
	if len(s.Fighters) > 0 {
		fighters = make([]posts.FighterNumber, 0, len(s.Fighters))
	}
	for _, number := range s.Fighters {
		fighters = append(fighters, posts.FighterNumber(number))
	}
	var ranks []posts.Rank
	if len(s.Ranks) > 0 {
		ranks = make([]posts.Rank, 0, len(s.Ranks))
	}
	for _, rank := range s.Ranks {
		ranks = append(ranks, posts.Rank{Code: rank.Code, Line: rank.Line})
	}
	// A time that is none is no time of a saga: the check refuses it.
	since, _ := time.Parse(time.RFC3339, s.Since)
	return posts.Story{
		Front: s.Front, Tour: s.Tour, Place: s.Place, Saga: s.Saga, Since: since, Kind: posts.Kind(s.Kind),
		Attacker: s.Attacker, Rival: s.Rival, TakenFrom: s.TakenFrom, Countries: s.Countries,
		Continent: s.Continent, Team: s.Team, Headline: s.Headline, Teasers: s.Teasers,
		Line: s.Line, Call: s.Call, CallFlags: s.CallFlags, Anthem: s.Anthem, Look: posts.Look(s.Look),
		Fighters: fighters, Hook: s.Hook, Language: s.Language, List: posts.ListName(s.List), Ranks: ranks,
	}
}

func cmpOr(value, otherwise string) string {
	if value == "" {
		return otherwise
	}
	return value
}
