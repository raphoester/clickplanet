package logging

import "github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"

// NewNopLogger returns a Logger that discards everything. Useful in tests and
// as a fallback when no logger was injected.
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

type NopLogger struct{}

func (NopLogger) Debug(string, ...lf.Field)   {}
func (NopLogger) Info(string, ...lf.Field)    {}
func (NopLogger) Warning(string, ...lf.Field) {}
func (NopLogger) Error(string, ...lf.Field)   {}
