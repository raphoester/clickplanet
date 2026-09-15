package messages_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func TestOverlongAuthorIDIsTruncated(t *testing.T) {
	record := messages.NewRecord(messages.Message{}, strings.Repeat("a", 5000), "1.2.3.4", "")
	assert.LessOrEqual(t, len(record.AuthorID), 64)
}

func TestOverlongUserAgentIsTruncated(t *testing.T) {
	record := messages.NewRecord(messages.Message{}, "", "1.2.3.4", strings.Repeat("a", 5000))
	assert.LessOrEqual(t, len(record.UserAgent), 256)
}
