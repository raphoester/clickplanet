package messages_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var (
	windowNow = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	window    = messages.Window{Size: 2, Retention: 24 * time.Hour}
)

func sentAgo(ago ...time.Duration) []messages.Message {
	shown := make([]messages.Message, 0, len(ago))
	for _, d := range ago {
		shown = append(shown, messages.Message{SentAt: windowNow.Add(-d)})
	}
	return shown
}

func TestAFullWindowBeginsAtItsOldestMessage(t *testing.T) {
	assert.Equal(t, windowNow.Add(-3*time.Hour), window.Beginning(windowNow, sentAgo(3*time.Hour, time.Hour)))
}

func TestAWindowWithRoomLeftBeginsAtTheRetention(t *testing.T) {
	assert.Equal(t, windowNow.Add(-24*time.Hour), window.Beginning(windowNow, sentAgo(time.Hour)),
		"nothing older was left out, so the history reaches as far back as it is kept")
}

func TestAnEmptyWindowBeginsAtTheRetention(t *testing.T) {
	assert.Equal(t, windowNow.Add(-24*time.Hour), window.Beginning(windowNow, nil))
}
