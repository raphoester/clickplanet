package cplogging

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging/cplf"

type Logger interface {
	Debug(message string, fields ...cplf.Field)
	Info(message string, fields ...cplf.Field)
	Warning(message string, fields ...cplf.Field)
	Error(message string, fields ...cplf.Field)
}
