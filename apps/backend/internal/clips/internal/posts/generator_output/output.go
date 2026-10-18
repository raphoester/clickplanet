package generator_output

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

const (
	clipSuffix  = ".json"
	videoSuffix = ".mp4"
)

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

type ShortlistFile string

func (f ShortlistFile) Shortlist(_ context.Context) (posts.Shortlist, error) {
	raw, err := os.ReadFile(string(f)) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return nil, fmt.Errorf("failed to read the shortlist: %w", err)
	}

	var written clipsv1.Shortlist
	if err := decoder.Unmarshal(raw, &written); err != nil {
		return nil, fmt.Errorf("the shortlist %s is not a clips.v1.Shortlist: %w", string(f), err)
	}

	stories := make([]posts.Story, 0, len(written.GetStories()))
	for _, story := range written.GetStories() {
		stories = append(stories, posts.Story{Title: story.GetTitle(), Key: keyOf(story.GetKey()), Countries: story.GetCountries()})
	}
	return posts.ShortlistOf(stories...) //nolint:wrapcheck // a pure delegation: the domain already named what is wrong.
}

type Folder string

var ErrNoVideo = errors.New("a clip has no video beside it")

func (f Folder) Rendered(_ context.Context) ([]posts.Clip, error) {
	described, err := filepath.Glob(filepath.Join(string(f), "*"+clipSuffix))
	if err != nil {
		return nil, fmt.Errorf("failed to list the clips: %w", err)
	}

	clips := make([]posts.Clip, 0, len(described))
	for _, path := range described {
		raw, err := os.ReadFile(path) //nolint:gosec // a file of the folder the operator of the job named.
		if err != nil {
			return nil, fmt.Errorf("failed to read a clip: %w", err)
		}

		var written clipsv1.Clip
		if err := decoder.Unmarshal(raw, &written); err != nil {
			return nil, fmt.Errorf("%s is not a clips.v1.Clip: %w", filepath.Base(path), err)
		}

		video := strings.TrimSuffix(path, clipSuffix) + videoSuffix
		if _, err := os.Stat(video); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrNoVideo, filepath.Base(video))
		}

		clip, err := posts.ClipSnapshot{
			Pick: posts.Pick(written.GetPick()), Title: written.GetTitle(), Key: keyOf(written.GetKey()),
			Teasers: written.GetTeasers(), Countries: countriesOf(written.GetCountries()),
			ContinentTag: written.GetContinentTag(), Attribution: written.GetCredit(), Video: posts.VideoFile(video),
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		clips = append(clips, clip)
	}
	return clips, nil
}

func keyOf(written *clipsv1.StoryKey) posts.StoryKey {
	return posts.StoryKey{
		Flag: written.GetFlag(), Continent: written.GetContinent(), Angle: posts.Angle(written.GetAngle()), Round: written.GetRound(),
	}
}

func countriesOf(written []*clipsv1.TaggedCountry) []posts.TaggedCountry {
	countries := make([]posts.TaggedCountry, 0, len(written))
	for _, country := range written {
		countries = append(countries, posts.TaggedCountry{Code: country.GetCode(), Tag: country.GetTag()})
	}
	return countries
}
