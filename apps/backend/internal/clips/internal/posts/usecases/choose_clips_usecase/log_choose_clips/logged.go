package log_choose_clips

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
)

func New(inner choose_clips_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  choose_clips_usecase.Executor
	logger *slog.Logger
}

var _ choose_clips_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (posts.Choice, error) {
	choice, err := l.inner.Execute(ctx)
	if err != nil {
		return choice, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	for _, story := range choice.Retold {
		l.logger.Info("skipped a story told already", attributesOf(story)...)
	}
	for _, story := range choice.Passed {
		l.logger.Info("passed a story for a flag or an angle told less lately", attributesOf(story)...)
	}
	switch {
	case choice.Room == 0:
		l.logger.Info("every platform has posted its day already")
	case len(choice.Chosen) == 0:
		l.logger.Info("no story is worth a clip")
	}
	l.logEditorial(choice.Editorial)
	for i, story := range choice.Chosen {
		attributes := attributesOf(story)
		if i < len(choice.Assigned) && len(choice.Assigned[i].Variants) > 0 {
			attributes = append(attributes, slog.String("arms", choice.Assigned[i].Variants.String()))
		}
		l.logger.Info("chose a story", attributes...)
	}

	return choice, nil
}

func (l *Logged) logEditorial(editorial posts.Editorial) {
	if editorial.Arm == "" {
		return
	}
	attributes := []any{slog.String("editor", editorial.Arm)}
	if editorial.Asked {
		spend := editorial.Spend
		attributes = append(attributes,
			slog.String("model", spend.Model), slog.Int64("input_tokens", spend.InputTokens),
			slog.Int64("cache_write_tokens", spend.CacheWriteTokens), slog.Int64("cache_read_tokens", spend.CacheReadTokens),
			slog.Int64("output_tokens", spend.OutputTokens), slog.String("usd", fmt.Sprintf("%.5f", spend.USD)))
	}
	switch {
	case editorial.FellBack != "":
		l.logger.Warn("the model's run is made by the rules", append(attributes, slog.String("why", editorial.FellBack))...)
	case editorial.Asked:
		l.logger.Info("the model edited the run", append(attributes, slog.String("why", editorial.Why))...)
	default:
		l.logger.Info("the run is edited", attributes...)
	}
}

func attributesOf(story posts.Story) []any {
	attributes := []any{
		slog.Int("pick", int(story.Pick)), slog.String("title", story.Title),
		slog.String("flag", story.Key.Flag), slog.String("continent", story.Key.Continent),
	}
	if story.Key.Angle != posts.Deeds {
		attributes = append(attributes, slog.String("angle", string(story.Key.Angle)))
	}
	if story.Key.Round != "" {
		attributes = append(attributes, slog.String("round", story.Key.Round))
	}
	return append(attributes, slog.String("countries", strings.Join(story.Countries, ",")))
}
