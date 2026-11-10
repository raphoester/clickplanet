// Package claude_name_judge asks a Claude model which player names the game's posts may show.
package claude_name_judge

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// What the platforms forbid in a name, sent first.
//
//go:embed instructions.md
var Instructions string

const (
	// The cheapest model, at $0.10 and $0.50 a million tokens in and out: a call of 100 names is a few thousand.
	DefaultModel = "claude-haiku-5-5"

	// The names of one call: a run's first cast names every player of 72 hours.
	batch = 100
	// Room for its thinking and a line a name.
	maxTokens = 16000
	// The job waits on the cast over ssh.
	timeout = 2 * time.Minute
	retries = 1
)

var (
	ErrRefused  = errors.New("the model refused to judge the names")
	ErrNoAnswer = errors.New("the model gave no answer the judge can read")
)

type Judge struct {
	client *anthropic.Client
	model  string
}

func New(client *anthropic.Client, model string) *Judge {
	return &Judge{client: client, model: model}
}

// With no key: nothing is sent, and no name is shown.
type Unkeyed struct{}

func (Unkeyed) Judge(context.Context, []string) ([]verdicts.Judgement, error) {
	return nil, fmt.Errorf("%w: %w", verdicts.ErrNoJudge, cpanthropic.ErrNoKey)
}

var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

func (j *Judge) Judge(ctx context.Context, names []string) ([]verdicts.Judgement, error) {
	var judged []verdicts.Judgement
	for start := 0; start < len(names); start += batch {
		some, err := j.judgeBatch(ctx, names[start:min(start+batch, len(names))])
		if err != nil {
			return nil, err
		}
		judged = append(judged, some...)
	}
	return judged, nil
}

func (j *Judge) judgeBatch(ctx context.Context, names []string) ([]verdicts.Judgement, error) {
	prompt, err := PromptOf(names)
	if err != nil {
		return nil, err
	}
	message, err := j.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     j.model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: Instructions}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchema},
		},
	}, requestOptions...)
	if err != nil {
		return nil, fmt.Errorf("the model could not be asked: %w", cpanthropic.Redacted(err))
	}

	switch message.StopReason {
	case anthropic.StopReasonRefusal:
		// Haiku has no fallback model on the server: a refusal is the end of it.
		category := string(message.StopDetails.Category)
		if category == "" {
			category = "no category"
		}
		return nil, fmt.Errorf("%w (%s)", ErrRefused, category)
	case anthropic.StopReasonEndTurn:
	default:
		return nil, fmt.Errorf("%w: it stopped on %s", ErrNoAnswer, message.StopReason)
	}
	return judgementsOf(names, message.Content)
}

type numbered struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

// The names as JSON, each with its number from 1: a name may hold any character.
func PromptOf(names []string) (string, error) {
	listed := make([]numbered, 0, len(names))
	for i, name := range names {
		listed = append(listed, numbered{Number: i + 1, Name: name})
	}
	encoded, err := json.Marshal(listed)
	if err != nil {
		return "", fmt.Errorf("failed to write the names: %w", err)
	}
	return "The names to judge:\n" + string(encoded), nil
}

// What the model answers, as the schema of its structured output holds it to.
type answer struct {
	Names []struct {
		Number int    `json:"number"`
		Shown  bool   `json:"shown"`
		Reason string `json:"reason"`
	} `json:"names"`
}

var answerSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"names"},
	"properties": map[string]any{
		"names": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"number", "shown", "reason"},
				"properties": map[string]any{
					"number": map[string]any{"type": "integer"},
					"shown":  map[string]any{"type": "boolean"},
					"reason": map[string]any{"type": "string"},
				},
			},
		},
	},
}

// The answer is the one text block; thinking blocks come before it and are left. A number the model made up, or
// answered twice, counts once or not at all: a name it left out stays hidden.
func judgementsOf(names []string, content []anthropic.ContentBlockUnion) ([]verdicts.Judgement, error) {
	for i := range content {
		text, ok := content[i].AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}
		var answered answer
		if err := json.Unmarshal([]byte(text.Text), &answered); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoAnswer, err)
		}
		seen := cpcolls.NewSet[int]()
		var judged []verdicts.Judgement
		for _, line := range answered.Names {
			if line.Number < 1 || line.Number > len(names) || seen.Contains(line.Number) {
				continue
			}
			seen.Add(line.Number)
			judged = append(judged, verdicts.Judgement{Name: names[line.Number-1], Shown: line.Shown, Reason: line.Reason})
		}
		return judged, nil
	}
	return nil, fmt.Errorf("%w: no text", ErrNoAnswer)
}
