package cpmail_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

func TestAMessageIDIsKeptWithoutItsBracketsAndWrittenWithThem(t *testing.T) {
	for _, raw := range []string{"<a1.b2@mail.example.com>", " a1.b2@mail.example.com ", "<a1.b2@mail.example.com"} {
		id, err := cpmail.MessageIDOf(raw)

		require.NoError(t, err, raw)
		assert.Equal(t, cpmail.MessageID("a1.b2@mail.example.com"), id, raw)
		assert.Equal(t, "<a1.b2@mail.example.com>", id.Header(), raw)
	}
}

func TestAMessageIDAHeaderCannotCarryIsRefused(t *testing.T) {
	for _, raw := range []string{"", "<>", "a b@c", "a\r\nBcc: x@y", "<a>b@c>", "é@example.com", strings.Repeat("a", 990)} {
		_, err := cpmail.MessageIDOf(raw)

		assert.ErrorIs(t, err, cpmail.ErrInvalidMessageID, "%q", raw)
	}
}

func TestAConfigNamesItsSenderAndHowItDelivers(t *testing.T) {
	logged := cpmail.Config{Delivery: cpmail.DeliveryLog, From: "support@example.com", FromName: "ClickPlanet"}
	require.NoError(t, logged.Validate())
	assert.Equal(t, cpmail.Named{Address: "support@example.com", Name: "ClickPlanet"}, logged.Sender())

	for name, broken := range map[string]cpmail.Config{
		"no sender":         {Delivery: cpmail.DeliveryLog},
		"another delivery":  {Delivery: "smtp", From: "support@example.com"},
		"no cloudflare key": {Delivery: cpmail.DeliveryCloudflare, From: "support@example.com"},
	} {
		require.Error(t, broken.Validate(), name)
	}
	sent := cpmail.Config{
		Delivery: cpmail.DeliveryCloudflare, From: "support@example.com",
		Cloudflare: cpmail.CloudflareConfig{AccountID: "a", APIToken: "t"},
	}
	assert.NoError(t, sent.Validate())
}
