package marketing

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/brevo_audience"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var database = cppg.Config{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "marketing"}

func TestAModuleThatIsOffChecksNothing(t *testing.T) {
	assert.NoError(t, Config{}.Validate())
}

func TestALocalModuleNeedsADatabaseAndAWebhookSecret(t *testing.T) {
	require.NoError(t, Config{Enabled: true, Database: database, WebhookSecret: "s", Audience: AudienceConfig{Delivery: "log"}}.Validate())

	err := Config{Enabled: true, Audience: AudienceConfig{Delivery: "log"}}.Validate()

	assert.ErrorContains(t, err, "marketing.database")
	assert.ErrorContains(t, err, "marketing.webhookSecret is empty")
}

func TestBrevoNeedsItsKeyListAndTemplate(t *testing.T) {
	err := Config{Enabled: true, Database: database, WebhookSecret: "s", Audience: AudienceConfig{Delivery: "brevo"}}.Validate()

	assert.ErrorContains(t, err, "apiKey is empty")
	assert.ErrorContains(t, err, "listId is not set")
	assert.ErrorContains(t, err, "doiTemplateId is not set")

	assert.NoError(t, Config{Enabled: true, Database: database, WebhookSecret: "s", Audience: AudienceConfig{
		Delivery: "brevo", Brevo: brevo_audience.Config{APIKey: "k", ListID: 7, DOITemplateID: 12},
	}}.Validate())
}

func TestAnUnknownDeliveryIsRefused(t *testing.T) {
	err := Config{Enabled: true, Database: database, WebhookSecret: "s", Audience: AudienceConfig{Delivery: "smtp"}}.Validate()

	assert.ErrorContains(t, err, `marketing.audience.delivery "smtp"`)
}

func TestTheThrottleDefaultsToFiveThenOneAMinute(t *testing.T) {
	config := Config{}.withDefaults()

	assert.Equal(t, 5, config.RateLimiter.Burst)
	assert.InDelta(t, 1.0/60, config.RateLimiter.PerSecond, 1e-9)
}

func TestTheConfigNeverPrintsItsSecrets(t *testing.T) {
	config := Config{ //nolint:gosec // test values, not credentials.
		WebhookSecret: "the-webhook-secret", Audience: AudienceConfig{Brevo: brevo_audience.Config{APIKey: "the-api-key"}},
	}

	printed := fmt.Sprintf("%v %+v", config, struct{ Marketing Config }{config})

	assert.NotContains(t, printed, "the-webhook-secret")
	assert.NotContains(t, printed, "the-api-key")
}
