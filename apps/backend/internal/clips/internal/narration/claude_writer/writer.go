// Package claude_writer asks a Claude model to write what a war report's voice says (docs/video-lab.md, "The war
// report's bench").
package claude_writer

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

// What every tone keeps, sent first and cached; the tone's brief follows it.
//
//go:embed instructions.md
var Instructions string

const (
	// Room for its thinking, which counts, and a script of 80 words.
	maxTokens = 8000
	// A writer is one call a script: one that does not answer in 2 minutes leaves its script out.
	timeout = 2 * time.Minute
	retries = 1
)

var (
	ErrRefused  = errors.New("the model refused")
	ErrNoAnswer = errors.New("the model gave no answer that can be read")
)

var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

type Writer struct {
	client *anthropic.Client
}

func New(client *anthropic.Client) *Writer {
	return &Writer{client: client}
}

// A script of the story, in its tone, by the model, and what the call cost. A model refused, cut short or not read is
// an error with the cost kept.
func (w *Writer) Write(ctx context.Context, model narration.Model, request narration.Request) (narration.Script, posts.Spend, error) {
	prompt, err := PromptOf(request)
	if err != nil {
		return narration.Script{}, posts.Spend{}, err
	}

	message, err := w.client.Beta.Messages.New(ctx, ParamsOf(model, request.Tone, prompt), requestOptions...)
	if err != nil {
		return narration.Script{}, posts.Spend{Model: string(model)}, fmt.Errorf("the model could not be asked: %w", cpanthropic.Redacted(err))
	}
	spend := SpendOf(model, message.Usage)

	switch message.StopReason {
	case anthropic.BetaStopReasonEndTurn:
	case anthropic.BetaStopReasonRefusal:
		return narration.Script{}, spend, fmt.Errorf("%w (%s)", ErrRefused, cmpOr(string(message.StopDetails.Category), "no category"))
	default:
		return narration.Script{}, spend, fmt.Errorf("%w: it stopped on %s", ErrNoAnswer, message.StopReason)
	}
	script, err := scriptOf(message.Content)
	return script, spend, err
}

// The request as sent: the instructions cached, the tone after them, the effort high on every model so the models
// compare on their writing alone, and on Sonnet and Opus a refusal served again by the fallback Anthropic chooses.
func ParamsOf(model narration.Model, tone narration.Tone, prompt string) anthropic.BetaMessageNewParams {
	params := anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System: []anthropic.BetaTextBlockParam{
			{Text: Instructions, CacheControl: anthropic.NewBetaCacheControlEphemeralParam()},
			{Text: tone.Brief},
		},
		Messages: []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt))},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortHigh,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: answerSchema},
		},
	}
	if model != narration.Haiku {
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
		params.Fallbacks = anthropic.BetaFallbacksParamOfDefault()
	}
	return params
}

type prompt struct {
	Story    story           `json:"story"`
	Facts    json.RawMessage `json:"facts"`
	Day      int             `json:"day"`
	Episodes []string        `json:"episodes"`
}

type story struct {
	Kind      string   `json:"kind,omitempty"`
	Attacker  string   `json:"attacker,omitempty"`
	Rival     string   `json:"rival,omitempty"`
	TakenFrom []string `json:"takenFrom,omitempty"`
	Countries []string `json:"countries,omitempty"`
	Continent string   `json:"continent,omitempty"`
	Since     string   `json:"since,omitempty"`
	Headline  string   `json:"headline"`
	Line      string   `json:"line,omitempty"`
}

// What the model is sent, after its instructions and its tone.
func PromptOf(request narration.Request) (string, error) {
	if !json.Valid([]byte(request.Facts)) {
		return "", fmt.Errorf("%w: the story's facts are not JSON", ErrNoAnswer)
	}
	told := request.Story
	written := prompt{
		Story: story{
			Kind: string(told.Kind), Attacker: told.Attacker, Rival: told.Rival, TakenFrom: told.TakenFrom,
			Countries: told.Countries, Continent: told.Continent, Headline: told.Headline, Line: told.Line,
		},
		Facts: json.RawMessage(request.Facts), Day: request.Day, Episodes: request.Episodes,
	}
	if !told.Since.IsZero() {
		written.Story.Since = told.Since.UTC().Format(time.RFC3339)
	}
	if written.Episodes == nil {
		written.Episodes = []string{}
	}
	text, err := json.Marshal(written)
	if err != nil {
		return "", fmt.Errorf("failed to write the prompt: %w", err)
	}
	return "The story to tell, as JSON:\n\n" + string(text), nil
}

type answer struct {
	Why   string   `json:"why"`
	Title string   `json:"title"`
	Lines []string `json:"lines"`
}

var answerSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"why", "title", "lines"},
	"properties": map[string]any{
		"why":   map[string]any{"type": "string"},
		"title": map[string]any{"type": "string"},
		"lines": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
}

// The answer is the one text block; thinking blocks come before it and are left.
func scriptOf(content []anthropic.BetaContentBlockUnion) (narration.Script, error) {
	for i := range content {
		text, ok := content[i].AsAny().(anthropic.BetaTextBlock)
		if !ok {
			continue
		}
		var answered answer
		if err := json.Unmarshal([]byte(text.Text), &answered); err != nil {
			return narration.Script{}, fmt.Errorf("%w: %w", ErrNoAnswer, err)
		}
		script := narration.Script{Title: strings.TrimSpace(answered.Title), Why: strings.TrimSpace(answered.Why)}
		for _, line := range answered.Lines {
			if said := strings.Join(strings.Fields(line), " "); said != "" {
				script.Lines = append(script.Lines, said)
			}
		}
		if len(script.Lines) == 0 {
			return narration.Script{}, fmt.Errorf("%w: no line", ErrNoAnswer)
		}
		return script, nil
	}
	return narration.Script{}, fmt.Errorf("%w: no text", ErrNoAnswer)
}

// Dollars a million tokens in and out, as Anthropic lists them (2026-11). A cache write is 1.25 times the input, a
// read 0.1 times; Haiku's prompt over 100K tokens is billed 5 times both, which a script never reaches.
var rates = map[narration.Model][2]float64{
	narration.Haiku:  {0.10, 0.50},
	narration.Sonnet: {2, 10},
	narration.Opus:   {4, 20},
}

func SpendOf(model narration.Model, usage anthropic.BetaUsage) posts.Spend {
	rate := rates[model]
	in, out := rate[0], rate[1]
	usd := (float64(usage.InputTokens)*in + float64(usage.CacheCreationInputTokens)*in*1.25 +
		float64(usage.CacheReadInputTokens)*in*0.1 + float64(usage.OutputTokens)*out) / 1_000_000
	return posts.Spend{
		Model: string(model), InputTokens: usage.InputTokens, CacheWriteTokens: usage.CacheCreationInputTokens,
		CacheReadTokens: usage.CacheReadInputTokens, OutputTokens: usage.OutputTokens, USD: usd,
	}
}

func cmpOr(value, otherwise string) string {
	if value == "" {
		return otherwise
	}
	return value
}
