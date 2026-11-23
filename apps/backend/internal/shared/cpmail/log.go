package cpmail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
)

type Log struct {
	logger *slog.Logger
}

var _ Mailer = (*Log)(nil)

func NewLog(logger *slog.Logger) *Log {
	return &Log{logger: logger}
}

func (l *Log) Send(_ context.Context, mail Mail) (MessageID, error) {
	drawn := make([]byte, 16)
	if _, err := rand.Read(drawn); err != nil {
		return "", fmt.Errorf("failed to draw a message id: %w", err)
	}
	id := MessageID(hex.EncodeToString(drawn) + "@" + mail.From.Address.Domain())

	l.logger.Warn("a mail was logged, not sent", slog.String("to", string(mail.To)),
		slog.String("reply_to", string(mail.ReplyTo)), slog.Any("headers", mail.Headers),
		slog.String("subject", mail.Subject), slog.String("text", mail.Text), slog.String("message_id", id.Header()))
	return id, nil
}
