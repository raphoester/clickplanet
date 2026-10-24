package log_choose_clips

import (
	"context"
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
	for _, story := range choice.Chosen {
		l.logger.Info("chose a story", attributesOf(story)...)
	}

	return choice, nil
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
