package cprules

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// PlayLog is what LoggedPlay says, in the module's words: Taken at Warn with each set taken, Failed at Error with each
// play that failed.
type PlayLog struct {
	Taken  string
	Failed string
}

func NewLoggedPlay[M proto.Message](inner Playing[M], logger *slog.Logger, says PlayLog) *LoggedPlay[M] {
	return &LoggedPlay[M]{inner: inner, logger: logger, says: says}
}

type LoggedPlay[M proto.Message] struct {
	inner  Playing[M]
	logger *slog.Logger
	says   PlayLog
}

func (l *LoggedPlay[M]) Execute(ctx context.Context) (Snapshot[M], bool, error) {
	taken, took, err := l.inner.Execute(ctx)
	l.log(ctx, taken, took, err)
	return taken, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func (l *LoggedPlay[M]) Play(ctx context.Context, choice Choice) (Snapshot[M], bool, error) {
	taken, took, err := l.inner.Play(ctx, choice)
	l.log(ctx, taken, took, err)
	return taken, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func (l *LoggedPlay[M]) log(ctx context.Context, taken Snapshot[M], took bool, err error) {
	if took {
		written, _ := protojson.Marshal(taken.Rules)
		l.logger.Warn(l.says.Taken, slog.String("set", string(taken.Name)), slog.String("rules", string(written)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error(l.says.Failed, slog.Any("error", err))
	}
}
