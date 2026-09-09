package logging

import "github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"

func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

type NopLogger struct{}

func (NopLogger) Debug(string, ...lf.Field)   {}
func (NopLogger) Info(string, ...lf.Field)    {}
func (NopLogger) Warning(string, ...lf.Field) {}
func (NopLogger) Error(string, ...lf.Field)   {}
