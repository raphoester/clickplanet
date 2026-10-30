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

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
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

	config cps3.Config
	bucket *s3_attachment_bucket.Bucket
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

	s.config = cps3.Config{
		Endpoint: endpoint, Region: "us-east-1", Name: bucketName, AccessKeyID: keyID, SecretAccessKey: secret,
	}
	s.bucket = s.bucketOf(s.config, cptime.SystemClock{})
}

func (s *testSuite) bucketOf(config cps3.Config, clock cptime.Clock) *s3_attachment_bucket.Bucket {
	bucket, err := s3_attachment_bucket.New(config, lifetimes, clock, http.DefaultClient)
	s.Require().NoError(err)
	return bucket
}

func fileOf(media notes.MediaType, body []byte) notes.Attachment {
	return notes.Attachment{ID: notes.AttachmentID(uuid.New()), MediaType: media, Size: int64(len(body))}
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

func (s *testSuite) view(file notes.Attachment) (int, string, []byte) {
	link, err := s.bucket.ViewURL(file.ID.String())
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

func (s *testSuite) uploaded(media notes.MediaType, body []byte) notes.Attachment {
	file := fileOf(media, body)
	link, err := s.bucket.UploadURL(file)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, s.put(link, string(media), body))
	return file
}

func (s *testSuite) TestAFileGoesUpThroughItsLinkAndComesBackThroughItsViewLink() {
	photo, video := s.uploaded(notes.MediaPNG, png), s.uploaded(notes.MediaMP4, mp4)

	status, contentType, body := s.view(photo)
	s.Equal(http.StatusOK, status)
	s.Equal("image/png", contentType)
	s.Equal(png, body)

	status, contentType, body = s.view(video)
	s.Equal(http.StatusOK, status)
	s.Equal("video/mp4", contentType)
	s.Equal(mp4, body)
}

func (s *testSuite) TestALinkTakesNoOtherSizeAndNoOtherTypeThanItsFile() {
	file := fileOf(notes.MediaPNG, png)
	link, err := s.bucket.UploadURL(file)
	s.Require().NoError(err)

	s.Equal(http.StatusForbidden, s.put(link, "image/png", append(bytes.Clone(png), '!')), "one byte more")
	s.Equal(http.StatusForbidden, s.put(link, "image/png", png[:len(png)-1]), "one byte less")
	s.Equal(http.StatusForbidden, s.put(link, "text/html", png), "another type")

	status, _, _ := s.view(file)
	s.Equal(http.StatusNotFound, status, "nothing was kept")
}

func (s *testSuite) TestTheBucketSaysTheTypeAndTheSizeOfEachFileItHoldsInTheOrderAsked() {
	photo, video := s.uploaded(notes.MediaPNG, png), s.uploaded(notes.MediaMP4, mp4)

	held, err := s.bucket.Attachments(s.T().Context(), video.ID, photo.ID)

	s.Require().NoError(err)
	s.Equal([]notes.Attachment{video, photo}, held)
}

func (s *testSuite) TestAFileThatWasNeverUploadedIsNotHeld() {
	photo := s.uploaded(notes.MediaPNG, png)

	_, err := s.bucket.Attachments(s.T().Context(), photo.ID, fileOf(notes.MediaMP4, mp4).ID)

	s.Require().ErrorIs(err, notes.ErrNotUploaded)
	status, _, _ := s.view(fileOf(notes.MediaPNG, png))
	s.Equal(http.StatusNotFound, status)
}

func (s *testSuite) TestADeletedFileIsGoneAndTheOthersStay() {
	photo, video := s.uploaded(notes.MediaPNG, png), s.uploaded(notes.MediaMP4, mp4)

	s.Require().NoError(s.bucket.Delete(s.T().Context(), photo.ID, fileOf(notes.MediaGIF, png).ID))

	_, err := s.bucket.Attachments(s.T().Context(), photo.ID)
	s.Require().ErrorIs(err, notes.ErrNotUploaded)
	held, err := s.bucket.Attachments(s.T().Context(), video.ID)
	s.Require().NoError(err)
	s.Equal([]notes.Attachment{video}, held)
	s.Require().NoError(s.bucket.Delete(s.T().Context(), photo.ID), "a file deleted twice is deleted")
}

func (s *testSuite) TestABucketWithAnotherSecretDeletesNothing() {
	photo := s.uploaded(notes.MediaPNG, png)
	wrong := s.config
	wrong.SecretAccessKey = "another-secret"

	err := s.bucketOf(wrong, cptime.SystemClock{}).Delete(s.T().Context(), photo.ID)

	s.Require().ErrorContains(err, "403")
	_, err = s.bucket.Attachments(s.T().Context(), photo.ID)
	s.Require().NoError(err)
}

func (s *testSuite) TestALinkPastItsTimeIsRefused() {
	late := s.bucketOf(s.config, cptime.NewFixedClock(time.Now().Add(-time.Hour)))
	link, err := late.UploadURL(fileOf(notes.MediaPNG, png))
	s.Require().NoError(err)

	s.Equal(http.StatusForbidden, s.put(link, "image/png", png))
}

func (s *testSuite) TestABucketWithAnotherSecretUploadsViewsAndReadsNothing() {
	photo := s.uploaded(notes.MediaPNG, png)
	wrong := s.config
	wrong.SecretAccessKey = "another-secret"
	stranger := s.bucketOf(wrong, cptime.SystemClock{})

	link, err := stranger.UploadURL(fileOf(notes.MediaPNG, png))
	s.Require().NoError(err)
	s.Equal(http.StatusForbidden, s.put(link, "image/png", png))

	view, err := stranger.ViewURL(photo.ID.String())
	s.Require().NoError(err)
	res, err := http.Get(view) //nolint:noctx,gosec // a link this test just made.
	s.Require().NoError(err)
	s.Require().NoError(res.Body.Close())
	s.Equal(http.StatusForbidden, res.StatusCode)

	_, err = stranger.Attachments(s.T().Context(), photo.ID)
	s.Require().ErrorContains(err, "403")
	s.Require().NotErrorIs(err, notes.ErrNotUploaded)
}

func (s *testSuite) TestABucketThatCannotBeReachedSaysSo() {
	away := s.config
	away.Endpoint = "http://127.0.0.1:1"

	_, err := s.bucketOf(away, cptime.SystemClock{}).Attachments(s.T().Context(), fileOf(notes.MediaPNG, png).ID)

	s.Require().ErrorContains(err, "reach the bucket")
}
