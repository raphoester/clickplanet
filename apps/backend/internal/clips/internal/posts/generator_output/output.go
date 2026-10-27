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

type FindingsFile string

func (f FindingsFile) Findings(_ context.Context) (posts.Findings, error) {
	raw, err := os.ReadFile(string(f)) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return posts.Findings{}, fmt.Errorf("failed to read the findings: %w", err)
	}

	var written clipsv1.Findings
	if err := decoder.Unmarshal(raw, &written); err != nil {
		return posts.Findings{}, fmt.Errorf("the findings %s are not a clips.v1.Findings: %w", string(f), err)
	}

	findings := posts.Findings{
		Since: written.GetSince().AsTime(), Until: written.GetUntil().AsTime(),
		Names: written.GetNames(), Continents: map[string]string{},
	}
	for _, continent := range written.GetContinents() {
		findings.Continents[continent.GetName()] = continent.GetFlag()
	}
	for _, front := range written.GetFronts() {
		findings.Fronts = append(findings.Fronts, frontOf(front))
	}
	for _, tour := range written.GetTours() {
		findings.Tours = append(findings.Tours, posts.Tour{
			Flag: tour.GetFlag(), Since: tour.GetSince().AsTime(), Until: tour.GetUntil().AsTime(), Took: int(tour.GetTook()),
			ByContinent: countsOf(tour.GetByContinent()), TookFrom: countsOf(tour.GetTookFrom()), NextTook: int(tour.GetNextTook()),
			Solidity: tour.GetSolidity(), Bombs: int(tour.GetBombs()),
		})
	}
	for _, place := range written.GetPlaces() {
		findings.Places = append(findings.Places, posts.Place{
			Kind: posts.PlaceKind(place.GetKind()), Code: place.GetCode(), Continent: place.GetContinent(), Tiles: int(place.GetTiles()),
			Holder: place.GetHolder(), Held: int(place.GetHeld()), Share: place.GetShare(), Rivals: countsOf(place.GetRivals()),
			Changed: int(place.GetChanged()),
		})
	}
	for _, seasonal := range written.GetSeasonal() {
		facts, err := factsOf(seasonal.GetFacts())
		if err != nil {
			return posts.Findings{}, err
		}
		findings.Seasonal = append(findings.Seasonal, posts.Seasonal{
			Title: seasonal.GetTitle(), Key: keyOf(seasonal.GetKey()), Countries: seasonal.GetCountries(), Teasers: seasonal.GetTeasers(),
			Facts: facts,
		})
	}
	return posts.FindingsOf(findings) //nolint:wrapcheck // a pure delegation: the domain already named what is wrong.
}

func frontOf(written *clipsv1.Front) posts.Front {
	front := posts.Front{
		Scope: written.GetScope(), Since: written.GetSince().AsTime(), Until: written.GetUntil().AsTime(),
		TilesChanged: int(written.GetTilesChanged()), Captured: int(written.GetCaptured()),
		Bombs: int(written.GetBombs()), Spreads: int(written.GetSpreads()), Enclosures: int(written.GetEnclosures()), Score: written.GetScore(),
	}
	for _, country := range written.GetCountries() {
		front.Countries = append(front.Countries, posts.FrontCountry{
			Code: country.GetCode(), Continent: country.GetContinent(), Tiles: int(country.GetTiles()), Changed: int(country.GetChanged()),
			HoldersStart: countsOf(country.GetHoldersStart()), HoldersEnd: countsOf(country.GetHoldersEnd()), Takers: countsOf(country.GetTakers()),
		})
	}
	for _, flag := range written.GetFlags() {
		front.Flags = append(front.Flags, posts.FrontFlag{
			Code: flag.GetCode(), Continent: flag.GetContinent(), Took: int(flag.GetTook()), Filled: int(flag.GetFilled()), Lost: int(flag.GetLost()),
			TookFrom: countsOf(flag.GetTookFrom()), Solidity: flag.GetSolidity(),
		})
	}
	return front
}

func countsOf(written map[string]uint32) map[string]int {
	counts := make(map[string]int, len(written))
	for code, count := range written {
		counts[code] = int(count)
	}
	return counts
}

// A story as the generator's --story takes it: a clips.v1.Pick in protojson.
func StoryJSON(story posts.Story) ([]byte, error) {
	written, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(&clipsv1.Pick{
		Front: uint32(story.Front), Tour: uint32(story.Tour), Place: uint32(story.Place), Seasonal: uint32(story.Seasonal), //nolint:gosec // from 1, a few dozen at most.
		Kind: string(story.Kind), Attacker: story.Attacker, Rival: story.Rival, TakenFrom: story.TakenFrom,
		Countries: story.Countries, Continent: story.Continent, Team: story.Team, Headline: story.Headline,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write the story: %w", err)
	}
	return written, nil
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

		record, err := recordOf(written.GetRecord())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}

		video := strings.TrimSuffix(path, clipSuffix) + videoSuffix
		if _, err := os.Stat(video); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrNoVideo, filepath.Base(video))
		}

		clip, err := posts.ClipSnapshot{
			Pick: posts.Pick(written.GetPick()), Title: written.GetTitle(), Key: keyOf(written.GetKey()),
			Teasers: written.GetTeasers(), Countries: countriesOf(written.GetCountries()),
			ContinentTag: written.GetContinentTag(), Attribution: written.GetCredit(), Video: posts.VideoFile(video),
			TeaserNames: written.GetTeaserNames(), Record: record, Variants: written.GetRecord().GetVariants(),
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

// Kept as the generator wrote it, in protojson; none for a clip written before the generator kept one.
func recordOf(written *clipsv1.ClipRecord) (posts.ClipRecord, error) {
	if written == nil {
		return nil, nil
	}
	record, err := protojson.Marshal(written)
	if err != nil {
		return nil, fmt.Errorf("failed to keep what the clip was: %w", err)
	}
	return record, nil
}

// Kept as the generator wrote them, for the editor; none from a generator that writes none.
func factsOf(written *clipsv1.StoryFacts) (posts.StoryFacts, error) {
	if written == nil {
		return nil, nil
	}
	facts, err := protojson.Marshal(written)
	if err != nil {
		return nil, fmt.Errorf("failed to keep what a story is: %w", err)
	}
	return facts, nil
}

func countriesOf(written []*clipsv1.TaggedCountry) []posts.TaggedCountry {
	countries := make([]posts.TaggedCountry, 0, len(written))
	for _, country := range written {
		countries = append(countries, posts.TaggedCountry{Code: country.GetCode(), Tag: country.GetTag()})
	}
	return countries
}
