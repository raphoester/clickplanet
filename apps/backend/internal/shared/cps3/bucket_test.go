package cps3_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cps3/cps3test"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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
	var err error
	s.bucket, err = cps3.New(cps3test.StartServer(s.T(), "media").Config(), cptime.SystemClock{}, http.DefaultClient)
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
