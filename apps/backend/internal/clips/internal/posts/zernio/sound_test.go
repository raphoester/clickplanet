package zernio_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/zernio"
)

// The document's shape: rank 1 first, a track with its trending excerpt, one without, one with neither id, and a field
// added later.
const chart = `{"tracks":[
	{"id":"7300000000000000001","commercialMusicId":"cm-1","name":"Song One","artist":"Singer","durationSec":182,"rank":1,
	 "clip":{"id":"7400000000000000001","durationSec":30,"previewUrl":"https://cdn.example/1.mp3"},"popularity":99},
	{"id":"7300000000000000002","commercialMusicId":"cm-2","name":"Song Two","artist":"Band","durationSec":201,"rank":2},
	{"commercialMusicId":"cm-3","name":"Song Three","artist":"Duo","rank":3,"clip":{}}
]}`

func trending(t *testing.T, fake *zernioFake, country string) ([]posts.Track, error) {
	t.Helper()

	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	return service.Trending(t.Context(), country) //nolint:wrapcheck // the test reads the service's own error.
}

func TestTheTrendingChartIsReadForTheTikTokAccountInTheCountryEachTrackByItsExcerpt(t *testing.T) {
	fake := newFake()
	fake.music = answer{status: http.StatusOK, body: chart}

	tracks, err := trending(t, fake, "US")

	require.NoError(t, err)
	assert.Equal(t, []string{"GET /v1/accounts", "GET /v1/accounts/acc-1/tiktok/commercial-music"}, fake.calls)
	assert.Equal(t, url.Values{"countryCode": {"US"}}, fake.queries["GET /v1/accounts/acc-1/tiktok/commercial-music"])
	assert.Equal(t, "Bearer the-key", fake.headers["GET /v1/accounts/acc-1/tiktok/commercial-music"].Get("Authorization"))
	assert.Equal(t, []posts.Track{
		{ID: "7400000000000000001", Excerpt: true, Title: "Song One", Artist: "Singer", Rank: 1, Seconds: 30},
		{ID: "7300000000000000002", Title: "Song Two", Artist: "Band", Rank: 2, Seconds: 201},
		{Title: "Song Three", Artist: "Duo", Rank: 3},
	}, tracks)
}

func TestAChartZernioRefusesIsAnErrorThatSaysWhy(t *testing.T) {
	fake := newFake()
	fake.music = answer{status: http.StatusBadRequest, body: `{"error":"Not a business account","code":"tiktok_business_required"}`}

	_, err := trending(t, fake, "US")

	require.EqualError(t, err, "zernio answered 400 to the commercial music library: Not a business account")
}

func TestAChartWithNoTracksInItIsAnError(t *testing.T) {
	fake := newFake()
	fake.music = answer{status: http.StatusOK, body: `{}`}

	_, err := trending(t, fake, "US")

	require.Error(t, err)
}

func TestNoTikTokAccountOrTwoIsNoChartAndNoCallForIt(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusOK, body: `{"accounts":[{"_id":"acc-2","platform":"youtube","isActive":true}]}`}

	_, err := trending(t, fake, "US")

	require.ErrorContains(t, err, "0 connected tiktok accounts")
	assert.Equal(t, []string{"GET /v1/accounts"}, fake.calls)
}

var song = posts.Sound{
	Track:   posts.Track{ID: "7400000000000000001", Excerpt: true, Title: "Song One", Artist: "Singer", Rank: 4, Seconds: 30},
	Country: "US", TrackVolume: 100, OriginalVolume: 0,
}

func sentOf(t *testing.T, post posts.Post) map[string]any {
	t.Helper()

	fake := newFake()
	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	_, err = service.Schedule(t.Context(), post)
	require.NoError(t, err)
	return fake.sent
}

func targetOf(t *testing.T, sent map[string]any, platform string) map[string]any {
	t.Helper()

	for _, target := range sent["platforms"].([]any) { //nolint:forcetypeassert // the test fails on a panic too.
		if target := target.(map[string]any); target["platform"] == platform { //nolint:forcetypeassert // the same.
			return target
		}
	}
	t.Fatalf("no %s target", platform)
	return nil
}

func recordOf(sent map[string]any) map[string]any {
	return sent["metadata"].(map[string]any)["record"].(map[string]any)["post"].(map[string]any) //nolint:forcetypeassert // the same.
}

var (
	anthemRecord   = posts.ClipRecord(`{"format":"globe-replay","variants":{"sound":"anthem"}}`)
	trendingRecord = posts.ClipRecord(`{"format":"globe-replay","variants":{"sound":"trending"}}`)
	europe         = posts.StoryKey{Flag: "fr", Continent: "Europe"}
)

// The control sends what a clip in no experiment sends, its record aside, which carries its arm as the generator wrote
// it: no music, no volume, key for key.
func TestAClipInTheAnthemArmIsSentAsAClipInNone(t *testing.T) {
	before := sentOf(t, postIn(t, nil, europe, anthemRecord, "takes-over"))

	anthem := sentOf(t, postIn(t, posts.Variants{posts.SoundExperiment: posts.WithAnthem}, europe, anthemRecord, "takes-over"))

	assert.Equal(t, before, anthem)
	assert.Equal(t, tiktokTarget(t)["platformSpecificData"], targetOf(t, anthem, "tiktok")["platformSpecificData"])
	assert.NotContains(t, recordOf(anthem), "sound")
	assert.NotContains(t, recordOf(anthem), "variants")
}

func TestAClipWithATrendingTrackSendsItToTikTokAloneOverTheMutedVideoAndRecordsIt(t *testing.T) {
	variants := posts.Variants{posts.SoundExperiment: posts.WithTrending}
	before := sentOf(t, postIn(t, variants, europe, trendingRecord, "takes-over"))

	sent := sentOf(t, postIn(t, variants, europe, trendingRecord, "takes-over").WithSound(song))

	settings := tiktokTarget(t)["platformSpecificData"].(map[string]any) //nolint:forcetypeassert // the test's own map.
	settings["musicSoundInfo"] = map[string]any{"musicSoundId": "7400000000000000001", "musicSoundVolume": float64(100), "musicSoundStart": float64(0)}
	settings["videoOriginalSoundVolume"] = float64(0)
	assert.Equal(t, settings, targetOf(t, sent, "tiktok")["platformSpecificData"])
	assert.Equal(t, targetOf(t, before, "youtube"), targetOf(t, sent, "youtube"))
	assert.Equal(t, before["mediaItems"], sent["mediaItems"])
	// protojson leaves out a zero: no originalVolume is the anthem muted.
	assert.Equal(t, map[string]any{
		"id": "7400000000000000001", "idOf": "excerpt", "title": "Song One", "artist": "Singer", "rank": float64(4),
		"chartCountry": "US", "durationSeconds": float64(30), "trackVolume": float64(100),
	}, recordOf(sent)["sound"])
	assert.NotContains(t, recordOf(sent), "variants")

	recordOf(sent)["sound"], recordOf(before)["sound"] = nil, nil
	targetOf(t, sent, "tiktok")["platformSpecificData"] = nil
	targetOf(t, before, "tiktok")["platformSpecificData"] = nil
	assert.Equal(t, before, sent)
}

func TestAWholeTrackIsRecordedAsOne(t *testing.T) {
	whole := song
	whole.Track.ID, whole.Track.Excerpt = "7300000000000000001", false

	sent := sentOf(t, postIn(t, posts.Variants{posts.SoundExperiment: posts.WithTrending}, europe, trendingRecord, "").WithSound(whole))

	assert.Equal(t, "track", recordOf(sent)["sound"].(map[string]any)["idOf"]) //nolint:forcetypeassert // the same.
}

// It goes out as the control, the record saying which arm it was posted in: the clip's says the arm it was drawn for.
func TestAClipThatFellBackIsSentAsTheControlAndRecordsTheArmItWentOutIn(t *testing.T) {
	variants := posts.Variants{"length": "short", posts.SoundExperiment: posts.WithTrending}
	control := sentOf(t, postIn(t, posts.Variants{"length": "short", posts.SoundExperiment: posts.WithAnthem}, europe, trendingRecord, ""))

	sent := sentOf(t, postIn(t, variants, europe, trendingRecord, "").FellBack(posts.SoundExperiment, errors.New("no chart")))

	assert.Equal(t, map[string]any{"length": "short", "sound": "fell-back"}, recordOf(sent)["variants"])
	assert.NotContains(t, recordOf(sent), "sound")
	delete(recordOf(sent), "variants")
	assert.Equal(t, control, sent)
}
