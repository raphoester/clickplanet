package cps3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	image      = "ghcr.io/versity/versitygw:v1.8.0"
	port       = "7070"
	bucketName = "media"
	keyID      = "the-key-id"
	secret     = "the-secret-key"
)

var video = []byte("an mp4 of the season")

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	bucket *cps3.Bucket
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

	s.bucket, err = cps3.New(cps3.Config{
		Endpoint: endpoint, Region: "us-east-1", Name: bucketName, AccessKeyID: keyID, SecretAccessKey: secret,
	}, cptime.SystemClock{}, http.DefaultClient)
	s.Require().NoError(err)
}

type answer struct {
	status int
	header http.Header
	body   []byte
}

func (s *testSuite) send(method, link string, body []byte, contentType string) answer {
	req, err := http.NewRequestWithContext(s.T().Context(), method, link, bytes.NewReader(body))
	s.Require().NoError(err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() { _ = res.Body.Close() }()

	read, err := io.ReadAll(res.Body)
	s.Require().NoError(err)
	return answer{status: res.StatusCode, header: res.Header, body: read}
}

func (s *testSuite) uploaded(key string) {
	link, err := s.bucket.UploadLink(key, cps3.Object{ContentType: "video/mp4", Size: int64(len(video))}, time.Minute)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, s.send(http.MethodPut, link, video, "video/mp4").status)
}

func (s *testSuite) TestAnObjectUnderAKeyWithSlashesGoesUpAndIsHeadedAndViewed() {
	key := "clips/videos/" + s.T().Name()
	s.uploaded(key)

	held, err := s.bucket.Head(s.T().Context(), key)
	s.Require().NoError(err)
	s.Equal(cps3.Object{ContentType: "video/mp4", Size: int64(len(video))}, held)

	link, err := s.bucket.ViewLink(key, time.Minute)
	s.Require().NoError(err)
	viewed := s.send(http.MethodGet, link, nil, "")
	s.Equal(http.StatusOK, viewed.status)
	s.Equal(video, viewed.body)
}

func (s *testSuite) TestADownloadLinkTellsTheBrowserTheNameToSaveItUnder() {
	key := "clips/videos/download"
	s.uploaded(key)

	link, err := s.bucket.DownloadLink(key, "Season-0-flat.mp4", time.Minute)
	s.Require().NoError(err)
	downloaded := s.send(http.MethodGet, link, nil, "")

	s.Equal(http.StatusOK, downloaded.status)
	s.Equal(`attachment; filename=Season-0-flat.mp4`, downloaded.header.Get("Content-Disposition"))
}

func (s *testSuite) TestAnObjectNeverSentIsNotFoundAndADeletedOneIsGone() {
	_, err := s.bucket.Head(s.T().Context(), "clips/videos/never")
	s.Require().ErrorIs(err, cps3.ErrNotFound)

	key := "clips/videos/deleted"
	s.uploaded(key)
	s.Require().NoError(s.bucket.Delete(s.T().Context(), key))
	_, err = s.bucket.Head(s.T().Context(), key)
	s.Require().ErrorIs(err, cps3.ErrNotFound)
	s.Require().NoError(s.bucket.Delete(s.T().Context(), key), "deleted twice is deleted")
}
