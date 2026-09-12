package cplogging

import (
	"log/slog"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging/cplf"
)

// TODO: inject config
func NewSLogger() *SLogger {
	return &SLogger{
		logger: slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})),
	}
}

type SLogger struct {
	logger *slog.Logger
}

func (s *SLogger) WithFields(fields ...cplf.Field) Logger {
	return &SLogger{
		logger: s.logger.With(s.parseFields(fields)...),
	}
}

func (s *SLogger) parseFields(fields []cplf.Field) []any {
	ret := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		ret = append(ret, f.Key(), f.Value())
	}
	return ret
}

func (s *SLogger) Debug(message string, fields ...cplf.Field) {
	s.logger.Debug(message, s.parseFields(fields)...)
}

func (s *SLogger) Info(message string, fields ...cplf.Field) {
	s.logger.Info(message, s.parseFields(fields)...)
}

func (s *SLogger) Warning(message string, fields ...cplf.Field) {
	s.logger.Warn(message, s.parseFields(fields)...)
}

func (s *SLogger) Error(message string, fields ...cplf.Field) {
	s.logger.Error(message, s.parseFields(fields)...)
}
