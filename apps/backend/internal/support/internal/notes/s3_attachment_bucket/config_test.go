package s3_attachment_bucket_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/s3_attachment_bucket"
)

var (
	whole = s3_attachment_bucket.Config{
		Endpoint: "https://account.r2.example", Region: "auto", Name: "support",
		AccessKeyID: "the-key-id", SecretAccessKey: "the-secret-key",
	}
	noon = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	file = notes.Attachment{
		ID: notes.AttachmentID(uuid.MustParse("6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a01")), MediaType: notes.MediaPNG, Size: 400,
	}
)

func bucketOf(t *testing.T, config s3_attachment_bucket.Config) *s3_attachment_bucket.Bucket {
	t.Helper()

	bucket, err := s3_attachment_bucket.New(config,
		s3_attachment_bucket.Lifetimes{Upload: 30 * time.Minute, View: time.Hour}, cptime.NewFixedClock(noon), http.DefaultClient)
	require.NoError(t, err)
	return bucket
}

func TestAWholeConfigIsValidAndEachMissingPartIsNamed(t *testing.T) {
	require.NoError(t, whole.Validate())

	for name, broken := range map[string]func(*s3_attachment_bucket.Config){
		"endpoint":        func(c *s3_attachment_bucket.Config) { c.Endpoint = "" },
		"region":          func(c *s3_attachment_bucket.Config) { c.Region = "" },
		"name":            func(c *s3_attachment_bucket.Config) { c.Name = "" },
		"accessKeyId":     func(c *s3_attachment_bucket.Config) { c.AccessKeyID = "" },
		"secretAccessKey": func(c *s3_attachment_bucket.Config) { c.SecretAccessKey = "" },
	} {
		config := whole
		broken(&config)

		require.ErrorContains(t, config.Validate(), name)
		_, err := s3_attachment_bucket.New(config, s3_attachment_bucket.Lifetimes{}, cptime.SystemClock{}, http.DefaultClient)
		require.Error(t, err, name)
	}
}

func TestAnEndpointIsASchemeAndAHostAndNothingElse(t *testing.T) {
	for _, endpoint := range []string{
		"account.r2.example", "ftp://account.r2.example", "https://", "https://account.r2.example/support",
		"https://account.r2.example?x=1", "https://key:secret@account.r2.example", "://",
	} {
		config := whole
		config.Endpoint = endpoint

		require.ErrorContains(t, config.Validate(), "endpoint", endpoint)
	}

	for _, endpoint := range []string{"https://account.r2.example", "https://account.r2.example/", "http://127.0.0.1:9000"} {
		config := whole
		config.Endpoint = endpoint

		require.NoError(t, config.Validate(), endpoint)
	}
}

func TestTheSecretIsNotInWhatAConfigPrints(t *testing.T) {
	printed := fmt.Sprintf("%+v", struct{ Bucket s3_attachment_bucket.Config }{whole})

	assert.Contains(t, printed, "the-key-id")
	assert.NotContains(t, printed, "the-secret-key")
}

func TestAnUploadLinkNamesTheFileAndSignsItsTypeAndItsSize(t *testing.T) {
	link, err := bucketOf(t, whole).UploadURL(file)
	require.NoError(t, err)

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "account.r2.example", parsed.Host)
	assert.Equal(t, "/support/6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a01", parsed.Path)
	assert.Equal(t, "content-length;content-type;host", parsed.Query().Get("X-Amz-SignedHeaders"))
	assert.Equal(t, "1800", parsed.Query().Get("X-Amz-Expires"))
	assert.Equal(t, "20261013T120000Z", parsed.Query().Get("X-Amz-Date"))
	assert.Equal(t, "the-key-id/20261013/auto/s3/aws4_request", parsed.Query().Get("X-Amz-Credential"))
	assert.Len(t, parsed.Query().Get("X-Amz-Signature"), 64)
	assert.NotContains(t, link, "the-secret-key")

	bigger := file
	bigger.Size++
	other, err := bucketOf(t, whole).UploadURL(bigger)
	require.NoError(t, err)
	assert.NotEqual(t, link, other, "the size is in the signature")
}

func TestAViewLinkIsOfTheSameFileAsTheUploadLinkAndLastsItsOwnTime(t *testing.T) {
	bucket := bucketOf(t, whole)
	upload, err := bucket.UploadURL(file)
	require.NoError(t, err)

	view, err := bucket.ViewURL(file.ID.String())
	require.NoError(t, err)

	uploadLink, err := url.Parse(upload)
	require.NoError(t, err)
	viewLink, err := url.Parse(view)
	require.NoError(t, err)
	assert.Equal(t, uploadLink.Path, viewLink.Path)
	assert.Equal(t, uploadLink.Host, viewLink.Host)
	assert.Equal(t, "host", viewLink.Query().Get("X-Amz-SignedHeaders"))
	assert.Equal(t, "3600", viewLink.Query().Get("X-Amz-Expires"))
}

func TestWithNoBucketSetEveryLinkAndEveryReadIsAnError(t *testing.T) {
	none := bucketOf(t, s3_attachment_bucket.Config{})

	_, err := none.UploadURL(file)
	require.ErrorIs(t, err, s3_attachment_bucket.ErrNoBucket)
	_, err = none.ViewURL(file.ID.String())
	require.ErrorIs(t, err, s3_attachment_bucket.ErrNoBucket)
	_, err = none.Attachments(t.Context(), file.ID)
	require.ErrorIs(t, err, s3_attachment_bucket.ErrNoBucket)
	require.ErrorIs(t, none.Delete(t.Context(), file.ID), s3_attachment_bucket.ErrNoBucket)
	require.NoError(t, none.Delete(t.Context()), "no file asked, nothing to ask the bucket")

	held, err := none.Attachments(t.Context())
	require.NoError(t, err)
	assert.Empty(t, held, "no file asked, nothing to ask the bucket")
}
