package cps3_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	whole = cps3.Config{
		Endpoint: "https://account.r2.example", Region: "auto", Name: "media",
		AccessKeyID: "the-key-id", SecretAccessKey: "the-secret-key",
	}
	noon  = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	photo = cps3.Object{ContentType: "image/png", Size: 400}
)

const key = "support/6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a01"

func bucketOf(t *testing.T, config cps3.Config) *cps3.Bucket {
	t.Helper()

	bucket, err := cps3.New(config, cptime.NewFixedClock(noon), http.DefaultClient)
	require.NoError(t, err)
	return bucket
}

func parsed(t *testing.T, link string, err error) *url.URL {
	t.Helper()

	require.NoError(t, err)
	out, err := url.Parse(link)
	require.NoError(t, err)
	return out
}

func TestAWholeConfigIsValidAndEachMissingPartIsNamed(t *testing.T) {
	require.NoError(t, whole.Validate())

	for name, broken := range map[string]func(*cps3.Config){
		"endpoint":        func(c *cps3.Config) { c.Endpoint = "" },
		"region":          func(c *cps3.Config) { c.Region = "" },
		"name":            func(c *cps3.Config) { c.Name = "" },
		"accessKeyId":     func(c *cps3.Config) { c.AccessKeyID = "" },
		"secretAccessKey": func(c *cps3.Config) { c.SecretAccessKey = "" },
	} {
		config := whole
		broken(&config)

		require.ErrorContains(t, config.Validate(), name)
		_, err := cps3.New(config, cptime.SystemClock{}, http.DefaultClient)
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
	printed := fmt.Sprintf("%+v", struct{ Bucket cps3.Config }{whole})

	assert.Contains(t, printed, "the-key-id")
	assert.NotContains(t, printed, "the-secret-key")
}

func TestAnUploadLinkNamesTheKeyAndSignsItsTypeAndItsSize(t *testing.T) {
	link, err := bucketOf(t, whole).UploadLink(key, photo, 30*time.Minute)
	upload := parsed(t, link, err)

	assert.Equal(t, "https", upload.Scheme)
	assert.Equal(t, "account.r2.example", upload.Host)
	assert.Equal(t, "/media/"+key, upload.Path, "the slash of the key stays a slash")
	assert.Equal(t, "content-length;content-type;host", upload.Query().Get("X-Amz-SignedHeaders"))
	assert.Equal(t, "1800", upload.Query().Get("X-Amz-Expires"))
	assert.Equal(t, "20261013T120000Z", upload.Query().Get("X-Amz-Date"))
	assert.Equal(t, "the-key-id/20261013/auto/s3/aws4_request", upload.Query().Get("X-Amz-Credential"))
	assert.Len(t, upload.Query().Get("X-Amz-Signature"), 64)
	assert.NotContains(t, link, "the-secret-key")

	bigger := photo
	bigger.Size++
	other, err := bucketOf(t, whole).UploadLink(key, bigger, 30*time.Minute)
	require.NoError(t, err)
	assert.NotEqual(t, link, other, "the size is in the signature")
}

func TestAViewLinkIsOfTheSameObjectAsTheUploadLinkAndLastsItsOwnTime(t *testing.T) {
	bucket := bucketOf(t, whole)
	link, err := bucket.UploadLink(key, photo, 30*time.Minute)
	upload := parsed(t, link, err)
	link, err = bucket.ViewLink(key, time.Hour)
	view := parsed(t, link, err)

	assert.Equal(t, upload.Path, view.Path)
	assert.Equal(t, upload.Host, view.Host)
	assert.Equal(t, "host", view.Query().Get("X-Amz-SignedHeaders"))
	assert.Equal(t, "3600", view.Query().Get("X-Amz-Expires"))
}

func TestADownloadLinkSignsTheNameTheFileIsSavedUnder(t *testing.T) {
	bucket := bucketOf(t, whole)
	link, err := bucket.DownloadLink(key, "season flat.mp4", time.Hour)
	download := parsed(t, link, err)
	link, err = bucket.ViewLink(key, time.Hour)
	view := parsed(t, link, err)

	assert.Equal(t, view.Path, download.Path)
	assert.Equal(t, `attachment; filename="season flat.mp4"`, download.Query().Get("response-content-disposition"))
	assert.NotEqual(t, view.Query().Get("X-Amz-Signature"), download.Query().Get("X-Amz-Signature"))
}

func TestWithNoBucketSetEveryLinkAndEveryCallIsAnError(t *testing.T) {
	none := bucketOf(t, cps3.Config{})

	_, err := none.UploadLink(key, photo, time.Minute)
	require.ErrorIs(t, err, cps3.ErrNoBucket)
	_, err = none.ViewLink(key, time.Minute)
	require.ErrorIs(t, err, cps3.ErrNoBucket)
	_, err = none.DownloadLink(key, "a.mp4", time.Minute)
	require.ErrorIs(t, err, cps3.ErrNoBucket)
	_, err = none.Head(t.Context(), key)
	require.ErrorIs(t, err, cps3.ErrNoBucket)
	require.ErrorIs(t, none.Delete(t.Context(), key), cps3.ErrNoBucket)
}
