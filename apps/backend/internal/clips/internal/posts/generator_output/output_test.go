package generator_output_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/generator_output"
)

const (
	writtenShortlist = `{
  "stories": [
    {
      "title": "KOSOVO IS ATTACKING THE MIDDLE EAST",
      "key": {
        "flag": "xk",
        "continent": "Middle East"
      }
    },
    {
      "title": "FRANCE IS ATTACKING EUROPE",
      "key": {
        "flag": "fr",
        "continent": "Europe"
      }
    }
  ]
}
`
	writtenClip = `{
  "pick": 2,
  "title": "FRANCE IS ATTACKING EUROPE",
  "key": {
    "flag": "fr",
    "continent": "Europe"
  },
  "teasers": [
    "FRANCE TAKES OVER EUROPE",
    "EUROPE IS UNDER ATTACK FROM FRANCE"
  ],
  "countries": [
    {
      "code": "fr",
      "tag": "france"
    },
    {
      "code": "de",
      "tag": "germany"
    }
  ],
  "continentTag": "europe",
  "credit": "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0"
}
`
)

func write(t *testing.T, folder, name, content string) string {
	t.Helper()

	path := filepath.Join(folder, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestAShortlistIsTheStoriesTheGeneratorWroteBestFirst(t *testing.T) {
	file := write(t, t.TempDir(), "shortlist.json", writtenShortlist)

	shortlist, err := generator_output.ShortlistFile(file).Shortlist(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Shortlist{
		{Pick: 1, Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", Key: posts.StoryKey{Flag: "xk", Continent: "Middle East"}},
		{Pick: 2, Title: "FRANCE IS ATTACKING EUROPE", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}},
	}, shortlist)
}

func TestAShortlistWithNoStoryIsEmpty(t *testing.T) {
	file := write(t, t.TempDir(), "shortlist.json", "{}\n")

	shortlist, err := generator_output.ShortlistFile(file).Shortlist(t.Context())

	require.NoError(t, err)
	assert.Empty(t, shortlist)
}

func TestAShortlistThatIsMissingOrIsNotOneIsAnError(t *testing.T) {
	folder := t.TempDir()

	_, err := generator_output.ShortlistFile(filepath.Join(folder, "gone.json")).Shortlist(t.Context())
	require.Error(t, err)

	_, err = generator_output.ShortlistFile(write(t, folder, "shortlist.json", "saved 8 stories")).Shortlist(t.Context())
	require.ErrorContains(t, err, "is not a clips.v1.Shortlist")
}

func TestAFolderGivesEachClipWithItsVideo(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-2.json", writtenClip)
	video := write(t, folder, "clip-2.mp4", "the video")
	write(t, folder, "clip-2.txt", "the caption, for a person")

	clips, err := generator_output.Folder(folder).Rendered(t.Context())

	require.NoError(t, err)
	require.Len(t, clips, 1)
	assert.Equal(t, posts.ClipSnapshot{
		Pick:    2,
		Title:   "FRANCE IS ATTACKING EUROPE",
		Key:     posts.StoryKey{Flag: "fr", Continent: "Europe"},
		Teasers: []string{"FRANCE TAKES OVER EUROPE", "EUROPE IS UNDER ATTACK FROM FRANCE"},
		Countries: []posts.TaggedCountry{
			{Code: "fr", Tag: "france"},
			{Code: "de", Tag: "germany"},
		},
		ContinentTag: "europe",
		Attribution:  "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0",
		Video:        posts.VideoFile(video),
	}, clips[0].TakeSnapshot())
}

func TestAFolderWithNoClipGivesNone(t *testing.T) {
	clips, err := generator_output.Folder(t.TempDir()).Rendered(t.Context())

	require.NoError(t, err)
	assert.Empty(t, clips)
}

func TestAClipWithNoVideoBesideItIsAnError(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-2.json", writtenClip)

	_, err := generator_output.Folder(folder).Rendered(t.Context())

	require.ErrorIs(t, err, generator_output.ErrNoVideo)
	assert.ErrorContains(t, err, "clip-2.mp4")
}

func TestAClipTheGeneratorLeftAPartOutOfIsAnError(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-1.json", `{"pick": 1, "title": "FRANCE IS ATTACKING EUROPE", "key": {"flag": "fr", "continent": "Europe"}}`)
	write(t, folder, "clip-1.mp4", "the video")

	_, err := generator_output.Folder(folder).Rendered(t.Context())

	require.ErrorIs(t, err, posts.ErrInvalidClip)
	assert.ErrorContains(t, err, "clip-1.json")
}

func TestAFileThatIsNotAClipIsAnError(t *testing.T) {
	folder := t.TempDir()
	write(t, folder, "clip-1.json", "not json")

	_, err := generator_output.Folder(folder).Rendered(t.Context())

	require.ErrorContains(t, err, "clip-1.json is not a clips.v1.Clip")
}
