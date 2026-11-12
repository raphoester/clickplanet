// Package claude_translator asks a Claude model to translate keyed lines, with the game's glossary and tone.
package claude_translator

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
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
)

// How the game talks and how a line is written: the prose the owner tunes, sent first and cached with the glossary.
//
//go:embed instructions.md
var Instructions string

const (
	// The page's lines cost cents with it, and it writes a language like its speakers.
	DefaultModel = "claude-sonnet-5-5"

	// Enough lines to share the cached instructions, few enough that one answer stays well under maxTokens.
	batchSize = 60
	maxTokens = 32000
	timeout   = 5 * time.Minute
	retries   = 2
)

type Translator struct {
	client   *anthropic.Client
	model    string
	glossary translation.Glossary
}

var _ translation.Translator = (*Translator)(nil)

func New(client *anthropic.Client, model string, glossary translation.Glossary) *Translator {
	return &Translator{client: client, model: model, glossary: glossary}
}

var (
	ErrRefused  = errors.New("the model refused")
	ErrNoAnswer = errors.New("the model gave no answer that reads")
)

var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

func (t *Translator) Translate(ctx context.Context, target translation.Language, lines []translation.Line) (map[translation.Key]string, translation.Spend, error) {
	answers := map[translation.Key]string{}
	var spend translation.Spend
	system := t.systemOf(target)
	for start := 0; start < len(lines); start += batchSize {
		batch := lines[start:min(start+batchSize, len(lines))]
		answered, spent, err := t.translateBatch(ctx, system, target, batch)
		spend = spend.Plus(spent)
		if err != nil {
			return answers, spend, err
		}
		for key, text := range answered {
			answers[key] = text
		}
	}
	return answers, spend, nil
}

func (t *Translator) translateBatch(ctx context.Context, system []anthropic.TextBlockParam, target translation.Language, lines []translation.Line) (map[translation.Key]string, translation.Spend, error) {
	prompt, err := promptOf(target, lines)
	if err != nil {
		return nil, translation.Spend{}, err
	}
	message, err := t.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     t.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchema},
		},
	}, requestOptions...)
	if err != nil {
		return nil, translation.Spend{}, fmt.Errorf("the model could not be asked: %w", cpanthropic.Redacted(err))
	}
	spend := translation.Spend{InputTokens: message.Usage.InputTokens + message.Usage.CacheReadInputTokens + message.Usage.CacheCreationInputTokens, OutputTokens: message.Usage.OutputTokens}
	switch message.StopReason {
	case anthropic.StopReasonRefusal:
		return nil, spend, ErrRefused
	case anthropic.StopReasonEndTurn:
	default:
		return nil, spend, fmt.Errorf("%w: it stopped on %s", ErrNoAnswer, message.StopReason)
	}
	answers, err := answersOf(message.Content)
	return answers, spend, err
}

func (t *Translator) systemOf(target translation.Language) []anthropic.TextBlockParam {
	var glossary strings.Builder
	fmt.Fprintf(&glossary, "## The glossary, in %s\n\n", nameOf(target))
	for _, term := range t.glossary {
		fmt.Fprintf(&glossary, "- %s → %s (%s)\n", term.English, term.Words[target], term.Means)
	}
	return []anthropic.TextBlockParam{
		{Text: Instructions},
		{Text: glossary.String(), CacheControl: anthropic.NewCacheControlEphemeralParam()},
	}
}

func nameOf(target translation.Language) string {
	tag, err := language.Parse(string(target))
	if err != nil {
		return string(target)
	}
	return fmt.Sprintf("%s (%s)", display.English.Tags().Name(tag), target)
}

type sentLine struct {
	Key     translation.Key `json:"key"`
	English string          `json:"english"`
	Context string          `json:"context,omitempty"`
}

func promptOf(target translation.Language, lines []translation.Line) (string, error) {
	sent := make([]sentLine, 0, len(lines))
	for _, line := range lines {
		sent = append(sent, sentLine{Key: line.Key, English: line.English, Context: line.Context})
	}
	encoded, err := json.MarshalIndent(sent, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to write the lines: %w", err)
	}
	return fmt.Sprintf("Translate these %d lines into %s.\n\n%s", len(lines), nameOf(target), encoded), nil
}

type answer struct {
	Lines []struct {
		Key  translation.Key `json:"key"`
		Text string          `json:"text"`
	} `json:"lines"`
}

var answerSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"lines"},
	"properties": map[string]any{
		"lines": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"key", "text"},
				"properties": map[string]any{
					"key":  map[string]any{"type": "string"},
					"text": map[string]any{"type": "string"},
				},
			},
		},
	},
}

// The answer is the one text block; thinking blocks come before it and are left.
func answersOf(content []anthropic.ContentBlockUnion) (map[translation.Key]string, error) {
	for i := range content {
		text, ok := content[i].AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}
		var answered answer
		if err := json.Unmarshal([]byte(text.Text), &answered); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoAnswer, err)
		}
		answers := make(map[translation.Key]string, len(answered.Lines))
		for _, line := range answered.Lines {
			answers[line.Key] = line.Text
		}
		return answers, nil
	}
	return nil, fmt.Errorf("%w: no text", ErrNoAnswer)
}
