package messages_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func TestOverlongUserAgentIsTruncated(t *testing.T) {
	record := messages.NewRecord(messages.Message{}, "1.2.3.4", strings.Repeat("a", 5000))
	assert.Len(t, record.TakeSnapshot().UserAgent, 256)
}

func TestAUserAgentIsKeptAsValidUTF8(t *testing.T) {
	for name, agent := range map[string]string{
		"cut inside a character": strings.Repeat("a", 255) + "é",
		"a raw byte":             "agent\xff/1.0",
	} {
		t.Run(name, func(t *testing.T) {
			kept := messages.NewRecord(messages.Message{}, "1.2.3.4", agent).TakeSnapshot().UserAgent

			assert.True(t, utf8.ValidString(kept))
			assert.LessOrEqual(t, len(kept), 256)
		})
	}
}
