package cps3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The example of "Authenticating Requests: Using Query Parameters (AWS Signature Version 4)", in the S3 reference.
func TestALinkIsSignedAsTheS3ReferenceSignsItsExample(t *testing.T) {
	//nolint:gosec // the key the reference prints.
	example := credentials{keyID: "AKIAIOSFODNN7EXAMPLE", secret: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", region: "us-east-1"}

	query := example.signedQuery(request{
		method: "GET", host: "examplebucket.s3.amazonaws.com", path: "/test.txt",
		at: time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC), ttl: 24 * time.Hour,
	})

	assert.Equal(t, "X-Amz-Algorithm=AWS4-HMAC-SHA256"+
		"&X-Amz-Credential=AKIAIOSFODNN7EXAMPLE%2F20130524%2Fus-east-1%2Fs3%2Faws4_request"+
		"&X-Amz-Date=20130524T000000Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host"+
		"&X-Amz-Signature=aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404", query)
}

func TestTheHeadersALinkSignsAreNamedInOrderAndChangeItsSignature(t *testing.T) {
	example := credentials{keyID: "a-key", secret: "a-secret", region: "auto"}
	upload := request{
		method: "PUT", host: "bucket.example", path: "/notes/a/0",
		headers: map[string]string{"content-type": "image/png", "content-length": "400"},
		at:      time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC), ttl: 30 * time.Minute,
	}
	bigger := upload
	bigger.headers = map[string]string{"content-type": "image/png", "content-length": "401"}

	query := example.signedQuery(upload)

	assert.Contains(t, query, "&X-Amz-Expires=1800&X-Amz-SignedHeaders=content-length%3Bcontent-type%3Bhost&")
	assert.Equal(t, query, example.signedQuery(upload), "the same request is the same link")
	assert.NotEqual(t, query, example.signedQuery(bigger))
}

func TestOnlyUnreservedCharactersAreLeftAsTheyAre(t *testing.T) {
	assert.Equal(t, "Az09-._~", escaped("Az09-._~"))
	assert.Equal(t, "a%2Fb%3Bc%20d%2B%C3%A9", escaped("a/b;c d+é"))
}

func TestAQueryParameterIsSignedInItsPlaceAmongTheOthers(t *testing.T) {
	example := credentials{keyID: "a-key", secret: "a-secret", region: "auto"}
	view := request{
		method: "GET", host: "bucket.example", path: "/media/clips/videos/a",
		at: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC), ttl: time.Hour,
	}
	download := view
	download.query = map[string]string{"response-content-disposition": `attachment; filename="a.mp4"`}

	query := example.signedQuery(download)

	assert.Contains(t, query, "&X-Amz-SignedHeaders=host&response-content-disposition=attachment%3B%20filename%3D%22a.mp4%22&X-Amz-Signature=")
	assert.NotEqual(t, example.signedQuery(view), query)
}
