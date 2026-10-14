package s3_attachment_bucket_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/s3_attachment_bucket"
)

const (
	image      = "ghcr.io/versity/versitygw:v1.8.0"
	port       = "7070"
	bucketName = "notes"
	keyID      = "the-key-id"
	secret     = "the-secret-key"
)

var lifetimes = s3_attachment_bucket.Lifetimes{Upload: 30 * time.Minute, View: time.Hour}

var (
	png = []byte("a png")
	mp4 = []byte("an mp4, longer")
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	config s3_attachment_bucket.Config
	bucket *s3_attachment_bucket.Bucket
	notes  byte
}

func (s *testSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(s.T().Context(), 3*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{port + "/tcp"},
			// A directory under the root is a bucket: no client is needed to make one.
			Entrypoint: []string{"sh", "-c", "mkdir -p /data/" + bucketName +
				" && exec versitygw --access " + keyID + " --secret " + secret + " --port :" + port + " posix /data"},
			WaitingFor: wait.ForListeningPort(port + "/tcp").WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	testcontainers.CleanupContainer(s.T(), container)
	s.Require().NoError(err, "failed to start the test bucket")

	endpoint, err := container.PortEndpoint(ctx, port, "http")
	s.Require().NoError(err)

	s.config = s3_attachment_bucket.Config{
		Endpoint: endpoint, Region: "us-east-1", Name: bucketName, AccessKeyID: keyID, SecretAccessKey: secret,
	}
	s.bucket = s.bucketOf(s.config, cptime.SystemClock{})
}

func (s *testSuite) bucketOf(config s3_attachment_bucket.Config, clock cptime.Clock) *s3_attachment_bucket.Bucket {
	bucket, err := s3_attachment_bucket.New(config, lifetimes, clock)
	s.Require().NoError(err)
	return bucket
}

func (s *testSuite) carrying(files ...[]byte) notes.Note {
	s.notes++
	snapshot := notes.Snapshot{
		ID: notes.NoteID(uuid.New()), Kind: notes.KindBug, Account: notes.AccountID{15: s.notes},
		Text: "the globe is black", At: time.Now(),
	}
	for n, file := range files {
		media := notes.MediaPNG
		if n%2 == 1 {
			media = notes.MediaMP4
		}
		snapshot.Attachments = append(snapshot.Attachments, notes.Attachment{MediaType: media, Size: int64(len(file))})
	}

	note, err := snapshot.Restore()
	s.Require().NoError(err)
	return note
}

func (s *testSuite) put(link string, contentType string, body []byte) int {
	req, err := http.NewRequestWithContext(s.T().Context(), http.MethodPut, link, bytes.NewReader(body))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", contentType)

	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() { _ = res.Body.Close() }()
	return res.StatusCode
}

func (s *testSuite) view(note notes.Note, position int) (int, string, []byte) {
	link, err := s.bucket.ViewURL(uuid.UUID(note.TakeSnapshot().ID).String(), position)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(s.T().Context(), http.MethodGet, link, nil)
	s.Require().NoError(err)
	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	s.Require().NoError(err)
	return res.StatusCode, res.Header.Get("Content-Type"), body
}

func (s *testSuite) uploaded(files ...[]byte) notes.Note {
	note := s.carrying(files...)
	links, err := s.bucket.UploadURLs(note)
	s.Require().NoError(err)
	s.Require().Len(links, len(files))

	for n, file := range files {
		s.Require().Equal(http.StatusOK, s.put(links[n], string(note.TakeSnapshot().Attachments[n].MediaType), file))
	}
	return note
}

func (s *testSuite) TestEachFileGoesUpThroughItsLinkAndComesBackThroughItsViewLink() {
	note := s.uploaded(png, mp4)

	status, contentType, body := s.view(note, 0)
	s.Equal(http.StatusOK, status)
	s.Equal("image/png", contentType)
	s.Equal(png, body)

	status, contentType, body = s.view(note, 1)
	s.Equal(http.StatusOK, status)
	s.Equal("video/mp4", contentType)
	s.Equal(mp4, body)
}

func (s *testSuite) TestALinkTakesNoOtherSizeAndNoOtherTypeThanItsAttachment() {
	note := s.carrying(png)
	links, err := s.bucket.UploadURLs(note)
	s.Require().NoError(err)

	s.Equal(http.StatusForbidden, s.put(links[0], "image/png", append(bytes.Clone(png), '!')), "one byte more")
	s.Equal(http.StatusForbidden, s.put(links[0], "image/png", png[:len(png)-1]), "one byte less")
	s.Equal(http.StatusForbidden, s.put(links[0], "text/html", png), "another type")

	status, _, _ := s.view(note, 0)
	s.Equal(http.StatusNotFound, status, "nothing was kept")
}

func (s *testSuite) TestTheViewLinkOfAFileThatWasNeverUploadedAnswersNotFound() {
	status, _, _ := s.view(s.carrying(png), 0)

	s.Equal(http.StatusNotFound, status)
}

func (s *testSuite) TestALinkPastItsTimeIsRefused() {
	late := s.bucketOf(s.config, cptime.NewFixedClock(time.Now().Add(-time.Hour)))
	note := s.carrying(png)
	links, err := late.UploadURLs(note)
	s.Require().NoError(err)

	s.Equal(http.StatusForbidden, s.put(links[0], "image/png", png))
}

func (s *testSuite) TestABucketWithAnotherSecretUploadsAndViewsNothing() {
	note := s.uploaded(png)
	wrong := s.config
	wrong.SecretAccessKey = "another-secret"
	stranger := s.bucketOf(wrong, cptime.SystemClock{})

	links, err := stranger.UploadURLs(note)
	s.Require().NoError(err)
	s.Equal(http.StatusForbidden, s.put(links[0], "image/png", png))

	link, err := stranger.ViewURL(uuid.UUID(note.TakeSnapshot().ID).String(), 0)
	s.Require().NoError(err)
	res, err := http.Get(link) //nolint:noctx,gosec // a link this test just made.
	s.Require().NoError(err)
	s.Require().NoError(res.Body.Close())
	s.Equal(http.StatusForbidden, res.StatusCode)
}
