package embedded_disposable_domains_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/embedded_disposable_domains"
)

func loaded(t *testing.T) *embedded_disposable_domains.Blocklist {
	t.Helper()

	blocklist := embedded_disposable_domains.New()
	require.NoError(t, blocklist.Load())
	return blocklist
}

func TestTheVendoredListLoadsWhole(t *testing.T) {
	assert.Greater(t, loaded(t).Size(), 8000)
}

func TestAWellKnownDisposableDomainIsListed(t *testing.T) {
	blocklist := loaded(t)

	for _, domain := range []string{"mailinator.com", "yopmail.com", "guerrillamail.com", "10minutemail.com"} {
		assert.True(t, blocklist.Disposable(domain), domain)
	}
}

func TestASubdomainOfADisposableDomainIsDisposable(t *testing.T) {
	assert.True(t, loaded(t).Disposable("inbox.mailinator.com"))
}

func TestAnEverydayProviderIsNotDisposable(t *testing.T) {
	blocklist := loaded(t)

	for _, domain := range []string{"gmail.com", "outlook.com", "proton.me", "yahoo.fr", "example.com", "com"} {
		assert.False(t, blocklist.Disposable(domain), domain)
	}
}

func TestTheCaseOfADomainDoesNotMatter(t *testing.T) {
	assert.True(t, loaded(t).Disposable("MailInator.COM"))
}
