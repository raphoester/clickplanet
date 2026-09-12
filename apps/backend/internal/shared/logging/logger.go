package logging

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/logging/lf"

type Logger interface {
	Debug(message string, fields ...lf.Field)
	Info(message string, fields ...lf.Field)
	Warning(message string, fields ...lf.Field)
	Error(message string, fields ...lf.Field)
}
