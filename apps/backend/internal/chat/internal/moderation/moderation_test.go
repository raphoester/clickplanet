package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
)

func TestEachKindIsAValueOfThePostgresType(t *testing.T) {
	for kind, kept := range map[moderation.Kind]string{
		moderation.Muted: "mute", moderation.Unmuted: "unmute", moderation.Removed: "removal",
	} {
		assert.Equal(t, kept, kind.String())
	}
}
