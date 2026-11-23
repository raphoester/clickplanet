package cpmail

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Named struct {
	Address Address
	Name    string
}

type Mail struct {
	From    Named
	To      Address
	ReplyTo Address
	Subject string
	Text    string
	HTML    string
	Headers map[string]string
}

type Mailer interface {
	Send(ctx context.Context, mail Mail) (MessageID, error)
}

var ErrInvalidMessageID = errors.New("this is not a message id")

// RFC 5322's line limit, less the angle brackets and "Message-ID: ".
const maxMessageIDLength = 984

// A Message-ID header's value, without its angle brackets.
type MessageID string

func MessageIDOf(raw string) (MessageID, error) {
	id := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(raw), "<"), ">")
	switch {
	case id == "":
		return "", fmt.Errorf("%w: it is empty", ErrInvalidMessageID)
	case len(id) > maxMessageIDLength:
		return "", fmt.Errorf("%w: it is %d bytes, over %d", ErrInvalidMessageID, len(id), maxMessageIDLength)
	case strings.ContainsFunc(id, func(r rune) bool {
		return r > unicode.MaxASCII || unicode.IsSpace(r) || unicode.IsControl(r) || r == '<' || r == '>'
	}):
		return "", fmt.Errorf("%w: %q holds a character a header cannot", ErrInvalidMessageID, id)
	}
	return MessageID(id), nil
}

func (id MessageID) Header() string {
	return "<" + string(id) + ">"
}
