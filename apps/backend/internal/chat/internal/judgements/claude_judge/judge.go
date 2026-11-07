// Package claude_judge asks a Claude model to judge a report of a player, against the moderation policy.
package claude_judge

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

// How the judge judges, sent first with the policy and cached.
//
//go:embed instructions.md
var Instructions string

const (
	// The cheapest model, at $0.10 and $0.50 a million tokens in and out: a judgement is a few thousand in.
	DefaultModel = "claude-haiku-5-5"

	// Room for its thinking and an answer of a few lines.
	maxTokens = 4000
	// A report waits on nobody: the bus tries again what fails.
	timeout = time.Minute
	retries = 1
	// The reported player's GIFs it is shown, the reported one first: each is an image the API fetches.
	maxImages = 3
)

var (
	// The model declined to judge: an operator's report.
	ErrRefused = fmt.Errorf("%w: the model refused", judgements.ErrUnjudgeable)
	// The API refused the request itself, which a try again would not change.
	ErrRejected = fmt.Errorf("%w: the API refused the request", judgements.ErrUnjudgeable)
	ErrNoAnswer = errors.New("the model gave no answer the judge can read")
)

type Judge struct {
	client *anthropic.Client
	model  string
	policy string
}

func New(client *anthropic.Client, model string, policy string) *Judge {
	return &Judge{client: client, model: model, policy: policy}
}

// With no key: nothing is sent, and every report is left to the operator.
type Unkeyed struct{}

func (Unkeyed) Judge(context.Context, judgements.Report, judgements.Conversation) (judgements.Answer, error) {
	return judgements.Answer{}, fmt.Errorf("%w: %w", judgements.ErrNoJudge, cpanthropic.ErrNoKey)
}

var requestOptions = []option.RequestOption{option.WithRequestTimeout(timeout), option.WithMaxRetries(retries)}

func (j *Judge) Judge(
	ctx context.Context, report judgements.Report, conversation judgements.Conversation,
) (judgements.Answer, error) {
	message, err := j.client.Messages.New(ctx, j.paramsOf(report, conversation), requestOptions...)
	if err != nil {
		var failed *anthropic.Error
		if errors.As(err, &failed) && permanent(failed.StatusCode) {
			return judgements.Answer{}, fmt.Errorf("%w: %w", ErrRejected, cpanthropic.Redacted(err))
		}
		return judgements.Answer{}, fmt.Errorf("the model could not be asked: %w", cpanthropic.Redacted(err))
	}
	spend := judgements.Spend{
		Model: j.model, Input: message.Usage.InputTokens, CacheWrite: message.Usage.CacheCreationInputTokens,
		CacheRead: message.Usage.CacheReadInputTokens, Output: message.Usage.OutputTokens,
	}

	switch message.StopReason {
	case anthropic.StopReasonRefusal:
		// Haiku has no fallback model on the server: a refusal is the end of it.
		category := string(message.StopDetails.Category)
		if category == "" {
			category = "no category"
		}
		return judgements.Answer{Spend: spend}, fmt.Errorf("%w (%s)", ErrRefused, category)
	case anthropic.StopReasonEndTurn:
	default:
		return judgements.Answer{Spend: spend}, fmt.Errorf("%w: it stopped on %s", ErrNoAnswer, message.StopReason)
	}

	answer, err := answerOf(message.Content)
	answer.Spend = spend
	return answer, err
}

// A 4xx but a timeout, a conflict or a rate limit: the same request is refused again.
func permanent(status int) bool {
	return status >= 400 && status < 500 &&
		status != http.StatusRequestTimeout && status != http.StatusConflict && status != http.StatusTooManyRequests
}

func (j *Judge) paramsOf(report judgements.Report, conversation judgements.Conversation) anthropic.MessageNewParams {
	return anthropic.MessageNewParams{
		Model:     j.model,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: Instructions},
			{Text: "# The moderation policy\n\n" + j.policy, CacheControl: anthropic.NewCacheControlEphemeralParam()},
		},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(ContentOf(report, conversation)...)},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: answerSchema},
		},
	}
}

// What the model reads of a report: the prompt, then the reported player's GIFs as images.
func ContentOf(report judgements.Report, conversation judgements.Conversation) []anthropic.ContentBlockParamUnion {
	prompt, images := PromptOf(report, conversation)
	blocks := make([]anthropic.ContentBlockParamUnion, 0, 1+2*len(images))
	blocks = append(blocks, anthropic.NewTextBlock(prompt))
	for _, line := range images {
		blocks = append(blocks,
			anthropic.NewTextBlock(fmt.Sprintf("The GIF of [%s]:", line.ID)),
			anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: line.Gif.URL}))
	}
	return blocks
}

// The report and the conversation as text, every author a label and never a name, and the GIFs shown as images.
func PromptOf(report judgements.Report, conversation judgements.Conversation) (string, []judgements.Line) {
	labels := labelsOf(report, conversation)
	images := imagesOf(report, conversation)

	var b strings.Builder
	fmt.Fprintf(&b, "Report %s, for %s.\n", report.ID, reasonOf(report.Reason))
	if text := strings.TrimSpace(report.Text); text != "" {
		fmt.Fprintf(&b, "What the reporter wrote: %q\n", text)
	}
	if conversation.Reported != nil {
		fmt.Fprintf(&b, "The message reported: [%s].\n", conversation.Reported.ID)
	} else {
		b.WriteString("The report names no message: it is of the player.\n")
	}

	if conversation.Reported != nil {
		fmt.Fprintf(&b, "\nThe room %s around the message reported, oldest first:\n", roomOf(conversation.Reported.Room))
		for _, line := range conversation.Before {
			b.WriteString(lineOf(line, labels, images))
		}
		b.WriteString(lineOf(*conversation.Reported, labels, images))
		for _, line := range conversation.After {
			b.WriteString(lineOf(line, labels, images))
		}
	}
	b.WriteString("\nThe reported player's own latest messages, oldest first:\n")
	if len(conversation.Own) == 0 {
		b.WriteString("(none kept)\n")
	}
	for _, line := range conversation.Own {
		b.WriteString(lineOf(line, labels, images))
	}
	return b.String(), images
}

func labelsOf(report judgements.Report, conversation judgements.Conversation) map[messages.AccountID]string {
	labels := map[messages.AccountID]string{report.Reported: "REPORTED", report.Reporter: "REPORTER"}
	for _, line := range conversation.Lines() {
		if _, named := labels[line.Account]; !named {
			labels[line.Account] = fmt.Sprintf("P%d", len(labels)-1)
		}
	}
	return labels
}

func imagesOf(report judgements.Report, conversation judgements.Conversation) []judgements.Line {
	var candidates []judgements.Line
	if conversation.Reported != nil {
		candidates = append(candidates, *conversation.Reported)
	}
	for i := len(conversation.Own) - 1; i >= 0; i-- {
		candidates = append(candidates, conversation.Own[i])
	}
	var images []judgements.Line
	for _, line := range candidates {
		if len(images) == maxImages {
			break
		}
		shown := false
		for _, image := range images {
			shown = shown || image.ID == line.ID
		}
		if !shown && !line.Removed && !line.Gif.None() && line.Account == report.Reported {
			images = append(images, line)
		}
	}
	return images
}

func lineOf(line judgements.Line, labels map[messages.AccountID]string, images []judgements.Line) string {
	body := line.Text
	if !line.Gif.None() {
		body = "[GIF]"
		for _, image := range images {
			if image.ID == line.ID {
				body = "[GIF, shown below]"
			}
		}
	}
	removed := ""
	if line.Removed {
		removed = " (removed)"
	}
	return fmt.Sprintf("[%s] %s %s %s%s: %s\n",
		line.ID, line.SentAt.UTC().Format("15:04"), roomOf(line.Room), labels[line.Account], removed, body)
}

func roomOf(room messages.Room) string {
	if room == messages.World {
		return "world"
	}
	return "room:" + string(room)
}

func reasonOf(reason judgements.Reason) string {
	switch reason {
	case judgements.ReasonChatAbuse:
		return "abuse in the chat"
	case judgements.ReasonCheating:
		return "cheating"
	case judgements.ReasonBadName:
		return "a bad name"
	default:
		return "another reason"
	}
}

// What the model answers, as the schema of its structured output holds it to.
type answer struct {
	Category  string   `json:"category"`
	Severe    bool     `json:"severe"`
	Offending []string `json:"offending"`
	What      string   `json:"what"`
	Why       string   `json:"why"`
}

var answerSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"category", "severe", "offending", "what", "why"},
	"properties": map[string]any{
		"category":  map[string]any{"type": "string", "enum": categories()},
		"severe":    map[string]any{"type": "boolean"},
		"offending": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"what":      map[string]any{"type": "string"},
		"why":       map[string]any{"type": "string"},
	},
}

func categories() []string {
	names := make([]string, 0, len(judgements.Categories))
	for _, category := range judgements.Categories {
		names = append(names, string(category))
	}
	return names
}

// The answer is the one text block; thinking blocks come before it and are left.
func answerOf(content []anthropic.ContentBlockUnion) (judgements.Answer, error) {
	for i := range content {
		text, ok := content[i].AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}
		var answered answer
		if err := json.Unmarshal([]byte(text.Text), &answered); err != nil {
			return judgements.Answer{}, fmt.Errorf("%w: %w", ErrNoAnswer, err)
		}
		offending := make([]messages.MessageID, 0, len(answered.Offending))
		for _, id := range answered.Offending {
			offending = append(offending, messages.MessageID(id))
		}
		return judgements.Answer{
			Category: judgements.Category(answered.Category), Severe: answered.Severe, Offending: offending,
			What: answered.What, Why: answered.Why,
		}, nil
	}
	return judgements.Answer{}, fmt.Errorf("%w: no text", ErrNoAnswer)
}
