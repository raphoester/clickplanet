package cplogging

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging/cplf"

func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

type NopLogger struct{}

func (NopLogger) Debug(string, ...cplf.Field)   {}
func (NopLogger) Info(string, ...cplf.Field)    {}
func (NopLogger) Warning(string, ...cplf.Field) {}
func (NopLogger) Error(string, ...cplf.Field)   {}
