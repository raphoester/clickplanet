package postgres_video_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_videos_handler/videos_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/postgres_video_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_video_store.Store
	query *videos_query.PostgresQuery
}

type links struct{}

func (links) ViewURL(id videos.VideoID) (string, error) {
	return "https://bucket.example/view/" + id.String(), nil
}

func (links) DownloadURL(id videos.VideoID, fileName string) (string, error) {
	return "https://bucket.example/download/" + id.String() + "/" + fileName, nil
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "clips", migrations.FS)
	s.store = postgres_video_store.New(s.db)
	s.query = videos_query.NewPostgresQuery(s.db, links{})
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var noon = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func (s *testSuite) kept(id byte, name string, at time.Time) videos.VideoSnapshot {
	snapshot := videos.VideoSnapshot{
		File: videos.File{ID: videos.VideoID{id}, MediaType: "video/mp4", Size: int64(id) * 1000},
		Name: name, Caption: "caption of " + name, KeptAt: at,
	}
	s.Require().NoError(s.store.Keep(s.T().Context(), snapshot.Restore()))
	return snapshot
}

func (s *testSuite) TestTheVideosAreListedNewestFirstWithTheirLinks() {
	s.kept(1, "Season 0", noon)
	s.kept(2, "Final Battle", noon.Add(time.Hour))

	listed, err := s.query.Videos(s.T().Context())
	s.Require().NoError(err)

	s.Require().Len(listed.GetVideos(), 2)
	newest := listed.GetVideos()[0]
	s.Equal("Final Battle", newest.GetName())
	s.Equal("caption of Final Battle", newest.GetCaption())
	s.Equal("video/mp4", newest.GetContentType())
	s.Equal(uint64(2000), newest.GetSizeBytes())
	s.Equal(noon.Add(time.Hour), newest.GetKeptAt().AsTime())
	s.Equal("https://bucket.example/view/"+videos.VideoID{2}.String(), newest.GetViewUrl())
	s.Equal("https://bucket.example/download/"+videos.VideoID{2}.String()+"/Final-Battle.mp4", newest.GetDownloadUrl())
	s.Equal("Season 0", listed.GetVideos()[1].GetName())
}

func (s *testSuite) TestAVideoKeptAgainTakesItsNewNameAndCaption() {
	first := s.kept(1, "Season 0", noon)
	again := first
	again.Name, again.Caption, again.KeptAt = "Season 0, live now", "Season 1 is live", noon.Add(time.Hour)
	s.Require().NoError(s.store.Keep(s.T().Context(), again.Restore()))

	listed, err := s.query.Videos(s.T().Context())
	s.Require().NoError(err)
	s.Require().Len(listed.GetVideos(), 1)
	s.Equal("Season 0, live now", listed.GetVideos()[0].GetName())
	s.Equal("Season 1 is live", listed.GetVideos()[0].GetCaption())
	s.Equal(noon, listed.GetVideos()[0].GetKeptAt().AsTime(), "it keeps its place in the list")
}

func (s *testSuite) TestARemovedVideoIsNoLongerListed() {
	s.kept(1, "Season 0", noon)
	s.kept(2, "Final Battle", noon)

	s.Require().NoError(s.store.Remove(s.T().Context(), videos.VideoID{1}))

	listed, err := s.query.Videos(s.T().Context())
	s.Require().NoError(err)
	s.Require().Len(listed.GetVideos(), 1)
	s.Equal("Final Battle", listed.GetVideos()[0].GetName())
	s.Require().ErrorIs(s.store.Remove(s.T().Context(), videos.VideoID{1}), videos.ErrNoVideo)
}
