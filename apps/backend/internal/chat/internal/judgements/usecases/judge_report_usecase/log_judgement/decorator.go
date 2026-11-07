package log_judgement

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase"
)

func New(inner judge_report_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  judge_report_usecase.Executor
	logger *slog.Logger
}

var _ judge_report_usecase.Executor = (*Logged)(nil)

// Every judgement, and what it cost: at Warn what the operator should look at (child safety, self-harm, cheating, a
// doubt, a severe offence, one after a 24-hour mute, the cap spent), at Info the rest. A failure is the subscriber's
// log line, and the bus tries it again.
func (l *Logged) Execute(ctx context.Context, report judgements.Report) (judge_report_usecase.Out, error) {
	out, err := l.inner.Execute(ctx, report)
	if err != nil {
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	judged := out.Judgement.TakeSnapshot()
	attributes := []any{
		slog.String("report", report.ID.String()), slog.String("message", string(report.Message)),
		slog.String("account", report.Reported.String()), slog.String("verdict", judged.Verdict.String()),
		slog.String("reason", judged.Reason),
	}
	if out.Spend.Model != "" {
		attributes = append(attributes,
			slog.String("model", out.Spend.Model), slog.Int64("inputTokens", out.Spend.Input),
			slog.Int64("cacheWriteTokens", out.Spend.CacheWrite), slog.Int64("cacheReadTokens", out.Spend.CacheRead),
			slog.Int64("outputTokens", out.Spend.Output), slog.Float64("dollars", out.Spend.Dollars()))
	}

	switch {
	case out.Again:
		l.logger.Debug("chat report told again", attributes...)
	case judged.Flagged && judged.Verdict == judgements.LeftOpen:
		l.logger.Warn("chat report left to the operator", attributes...)
	case judged.Flagged:
		l.logger.Warn("chat report acted on, for the operator to see", attributes...)
	default:
		l.logger.Info("chat report judged", attributes...)
	}
	return out, nil
}
