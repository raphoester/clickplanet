package support_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support"
)

func validConfig() support.Config {
	config := support.Config{
		Database: cppg.Config{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "support"},
	}
	config.Attachments.Enabled = true
	config.Attachments.Bucket.Endpoint = "https://account.r2.example"
	config.Attachments.Bucket.Region = "auto"
	config.Attachments.Bucket.Name = "support"
	config.Attachments.Bucket.AccessKeyID = "the-key-id"
	config.Attachments.Bucket.SecretAccessKey = "the-secret-key"
	return config
}

func TestAConfigWithAWholeBucketIsValid(t *testing.T) {
	require.NoError(t, validConfig().Validate())
}

func TestAttachmentsOffCheckNothingOfTheBucket(t *testing.T) {
	config := support.Config{Database: validConfig().Database}
	config.Attachments.Bucket.Endpoint = "not a url"
	config.Attachments.LinkTTL = 30 * 24 * time.Hour

	assert.NoError(t, config.Validate())
}

func TestAttachmentsOnWithABucketMissingAPartAreRefused(t *testing.T) {
	for want, change := range map[string]func(*support.Config){
		"support.attachments.bucket: endpoint":           func(c *support.Config) { c.Attachments.Bucket.Endpoint = "" },
		"support.attachments.bucket: region is empty":    func(c *support.Config) { c.Attachments.Bucket.Region = "" },
		"support.attachments.bucket: name is empty":      func(c *support.Config) { c.Attachments.Bucket.Name = "" },
		"support.attachments.bucket: accessKeyId is":     func(c *support.Config) { c.Attachments.Bucket.AccessKeyID = "" },
		"support.attachments.bucket: secretAccessKey is": func(c *support.Config) { c.Attachments.Bucket.SecretAccessKey = "" },
	} {
		t.Run(want, func(t *testing.T) {
			config := validConfig()
			change(&config)

			assert.ErrorContains(t, config.Validate(), want)
		})
	}
}

func TestALinkThatLastsLongerThanABucketTakesIsRefused(t *testing.T) {
	week := 7 * 24 * time.Hour

	for want, change := range map[string]func(*support.Config){
		"support.attachments.uploadTTL": func(c *support.Config) { c.Attachments.UploadTTL = week + time.Second },
		"support.attachments.linkTTL":   func(c *support.Config) { c.Attachments.LinkTTL = week + time.Second },
	} {
		config := validConfig()
		change(&config)

		require.ErrorContains(t, config.Validate(), want)
	}

	config := validConfig()
	config.Attachments.UploadTTL, config.Attachments.LinkTTL = week, week
	require.NoError(t, config.Validate())
}

func TestAnIncompleteDatabaseIsRefusedWhateverTheAttachments(t *testing.T) {
	on := validConfig()
	on.Database.Host = ""
	off := support.Config{}

	require.ErrorContains(t, on.Validate(), "support.database")
	require.ErrorContains(t, off.Validate(), "support.database")
}
