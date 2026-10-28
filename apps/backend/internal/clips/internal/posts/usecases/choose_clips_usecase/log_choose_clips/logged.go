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

	switch {
	case choice.Room == 0:
		l.logger.Info("every platform has posted its day already")
	case len(choice.Chosen) == 0:
		l.logger.Info("no story is told this run")
	}
	l.logEditorial(choice)
	for _, skipped := range choice.Skipped {
		l.logger.Warn("skipped a story of the editor's", append(attributesOf(skipped.Story), slog.String("why", skipped.Why))...)
	}
	for i, chosen := range choice.Chosen {
		attributes := append(attributesOf(chosen.Story), slog.Int("pick", int(chosen.Pick)),
			slog.String("flag", chosen.Key.Flag), slog.String("continent", chosen.Key.Continent))
		if chosen.Key.Angle != posts.Deeds {
			attributes = append(attributes, slog.String("angle", string(chosen.Key.Angle)))
		}
		if chosen.Key.Round != "" {
			attributes = append(attributes, slog.String("round", chosen.Key.Round))
		}
		if i < len(choice.Assigned) && len(choice.Assigned[i].Variants) > 0 {
			attributes = append(attributes, slog.String("arms", choice.Assigned[i].Variants.String()))
		}
		l.logger.Info("chose a story", attributes...)
	}

	return choice, nil
}

// How the editor's call went: what it cost, its reason, or why no answer was had. A quiet run says nothing of it.
func (l *Logged) logEditorial(choice posts.Choice) {
	editorial := choice.Editorial
	var attributes []any
	if editorial.Asked {
		spend := editorial.Spend
		attributes = append(attributes,
			slog.String("model", spend.Model), slog.Int64("input_tokens", spend.InputTokens),
			slog.Int64("cache_write_tokens", spend.CacheWriteTokens), slog.Int64("cache_read_tokens", spend.CacheReadTokens),
			slog.Int64("output_tokens", spend.OutputTokens), slog.String("usd", fmt.Sprintf("%.5f", spend.USD)))
	}
	switch {
	case editorial.Failed != "":
		l.logger.Warn("the editor gave no answer, and the run is its seasonal stories alone", append(attributes, slog.String("why", editorial.Failed))...)
	case editorial.Asked:
		l.logger.Info("the editor made the run", append(attributes, slog.String("why", editorial.Why))...)
	case len(choice.Chosen) > 0:
		l.logger.Info("the editor had nothing to decide")
	}
}

func attributesOf(story posts.Story) []any {
	attributes := []any{slog.String("of", story.Names())}
	if story.Headline != "" {
		attributes = append(attributes, slog.String("headline", story.Headline))
	}
	if len(story.Teasers) > 0 {
		attributes = append(attributes, slog.String("teasers", strings.Join(story.Teasers, " / ")))
	}
	if story.Front > 0 {
		place := story.Continent
		if place == "" {
			place = strings.Join(story.Countries, "+")
		}
		attributes = append(attributes, slog.String("kind", string(story.Kind)), slog.String("attacker", story.Attacker),
			slog.String("from", strings.Join(story.TakenFrom, ",")), slog.String("in", place))
		if story.Rival != "" {
			attributes = append(attributes, slog.String("rival", story.Rival))
		}
		if story.Team != "" {
			attributes = append(attributes, slog.String("team", story.Team))
		}
	}
	return attributes
}
