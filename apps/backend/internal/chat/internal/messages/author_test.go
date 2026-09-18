package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func TestAnAccountIsReadOffTheContextsValue(t *testing.T) {
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", messages.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11").String())

	for _, value := range []string{"", "not-a-uuid"} {
		assert.Equal(t, cpsession.NoAccount, messages.AccountIDOf(value), "%q", value)
	}
}
