// Package claude_flag_judge asks a Claude model whether a regional flag sent for the game opens.
package claude_flag_judge

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

// The game's review rule, sent first.
//
//go:embed instructions.md
var Instructions string

const (
	// Claude Haiku 5.5, which reads pictures: a flag is about two thousand tokens in.
	DefaultModel = "claude-haiku-5-5"

	maxTokens = 4000
	// The buyer is on Stripe's page meanwhile.
	timeout = 45 * time.Second
	retries = 1
)

type Judge struct {
	client *anthropic.Client
	model  string
}

func New(client *anthropic.Client, model string) *Judge {
	if model == "" {
		model = DefaultModel
	}
	return &Judge{client: client, model: model}
}

var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

// Judge rules on a case. A refusal, or an answer the game cannot read, is unsure: the operator decides.
func (j *Judge) Judge(ctx context.Context, sent judgements.Case) (judgements.Ruling, error) {
	prompt, err := promptOf(sent)
	if err != nil {
		return judgements.Ruling{}, err
	}
	message, err := j.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     j.model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: Instructions}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(
			anthropic.NewImageBlockBase64("image/png", base64.StdEncoding.EncodeToString(sent.Picture)),
			anthropic.NewTextBlock(prompt),
		)},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchema},
		},
	}, requestOptions...)
	if err != nil {
		return judgements.Ruling{}, fmt.Errorf("the model could not be asked: %w", cpanthropic.Redacted(err))
	}
	switch message.StopReason {
	case anthropic.StopReasonRefusal:
		return judgements.Ruling{Verdict: judgements.Unsure, Why: "the model refused to judge it"}, nil
	case anthropic.StopReasonEndTurn:
	default:
		return judgements.Ruling{Verdict: judgements.Unsure, Why: "the model stopped on " + string(message.StopReason)}, nil
	}
	return rulingOf(message.Content), nil
}

type asked struct {
	Name    string   `json:"name"`
	Capital capital  `json:"capital"`
	Taken   []string `json:"flags_in_the_game"`
}

type capital struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// The case as JSON beside the picture: a name may hold any character.
func promptOf(sent judgements.Case) (string, error) {
	taken := make([]string, 0, len(sent.Taken))
	for _, name := range sent.Taken {
		taken = append(taken, name.String())
	}
	encoded, err := json.Marshal(asked{
		Name: sent.Name.String(), Capital: capital{Latitude: sent.Pin.Latitude, Longitude: sent.Pin.Longitude},
		Taken: taken,
	})
	if err != nil {
		return "", fmt.Errorf("failed to write the case: %w", err)
	}
	return "The flag sent, its picture above:\n" + string(encoded), nil
}

type answer struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
	Note    string `json:"note"`
	Why     string `json:"why"`
}

var answerSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"verdict", "reason", "note", "why"},
	"properties": map[string]any{
		"verdict": map[string]any{"type": "string", "enum": []string{"approve", "reject", "unsure"}},
		"reason": map[string]any{
			"type": "string", "enum": []string{"", "not_a_place", "exists", "unreadable", "hateful", "other"},
		},
		"note": map[string]any{"type": "string"},
		"why":  map[string]any{"type": "string"},
	},
}

// The answer is the one text block; thinking blocks come before it and are left.
func rulingOf(content []anthropic.ContentBlockUnion) judgements.Ruling {
	for i := range content {
		text, ok := content[i].AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}
		var answered answer
		if err := json.Unmarshal([]byte(text.Text), &answered); err != nil {
			return judgements.Ruling{Verdict: judgements.Unsure, Why: "an answer the game cannot read"}
		}
		return judgements.ParseRuling(answered.Verdict, answered.Reason, answered.Note, answered.Why)
	}
	return judgements.Ruling{Verdict: judgements.Unsure, Why: "no answer"}
}
