// Package claude_editor asks a Claude model to edit a run: which of the stories offered it tells, and their headlines.
package claude_editor

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

	// Room for its thinking, which counts, and a ranking of a dozen stories with their headlines.
	maxTokens = 8000
	// One call a run, in a job of 10 minutes: a model that does not answer in 2 leaves the run to the rules.
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

// The editor's own patience, on the client it is given.
var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

func (e *Editor) Edit(ctx context.Context, offer posts.Offer) (posts.Edition, posts.Spend, error) {
	prompt, err := PromptOf(offer)
	if err != nil {
		return posts.Edition{}, posts.Spend{}, err
	}

	message, err := e.client.Messages.New(ctx, e.paramsOf(prompt), requestOptions...)
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

func (e *Editor) paramsOf(prompt string) anthropic.MessageNewParams {
	return anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: Instructions, CacheControl: anthropic.NewCacheControlEphemeralParam()}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchema},
		},
	}
}

// The prompt's size, as the model counts it, with no answer asked for: what `clips edit -prompt` shows.
func (e *Editor) Count(ctx context.Context, offer posts.Offer) (int64, error) {
	prompt, err := PromptOf(offer)
	if err != nil {
		return 0, err
	}
	params := e.paramsOf(prompt)
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
	Why     string `json:"why"`
	Stories []struct {
		Pick     int    `json:"pick"`
		Headline string `json:"headline"`
	} `json:"stories"`
}

var answerSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"why", "stories"},
	"properties": map[string]any{
		"why": map[string]any{"type": "string"},
		"stories": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"pick", "headline"},
				"properties": map[string]any{
					"pick":     map[string]any{"type": "integer"},
					"headline": map[string]any{"type": "string"},
				},
			},
		},
	},
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
			edition.Edits = append(edition.Edits, posts.Edit{Pick: posts.Pick(story.Pick), Headline: story.Headline})
		}
		return edition, nil
	}
	return posts.Edition{}, fmt.Errorf("%w: no text", ErrNoAnswer)
}

func cmpOr(value, otherwise string) string {
	if value == "" {
		return otherwise
	}
	return value
}
