// What a clip's post keeps with it, never published: written when it is scheduled, read back by the next runs and by
// the snapshot of its numbers. The publishing module keeps it as it is given.
package metadata

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

const (
	flagKey      = "storyFlag"
	continentKey = "storyContinent"
	countriesKey = "storyCountries"
	callKey      = "callToAction"
	// Absent on a post of what a flag did, and on every post made before them.
	angleKey = "storyAngle"
	roundKey = "storyRound"
	// What the post was, to set beside how it did: a clips.v1.PostRecord.
	recordKey = "record"
)

// What the choice reads back of the post: its key and its countries; and the call to act, to compare the calls on the
// posts' numbers. Beside them, what the post was, as one document (recordKey), with the time it was planned at.
func Of(snapshot posts.PostSnapshot) (map[string]any, error) {
	record, err := recordOf(snapshot)
	if err != nil {
		return nil, err
	}
	metadata := map[string]any{
		flagKey: snapshot.Key.Flag, continentKey: snapshot.Key.Continent, countriesKey: snapshot.Countries,
		callKey: snapshot.CallToAction, recordKey: record,
	}
	if snapshot.Key.Angle != posts.Deeds {
		metadata[angleKey] = string(snapshot.Key.Angle)
	}
	if snapshot.Key.Round != "" {
		metadata[roundKey] = snapshot.Key.Round
	}
	return metadata, nil
}

// A clips.v1.PostRecord as protojson writes it, under one key: what the clip was as the generator wrote it, and what
// this job made of it. Its shape is the proto's, and it grows there.
func recordOf(snapshot posts.PostSnapshot) (map[string]any, error) {
	record := &clipsv1.PostRecord{
		Schema: string((&clipsv1.PostRecord{}).ProtoReflect().Descriptor().FullName()),
		Post: &clipsv1.PostFacts{
			Teaser: snapshot.Teaser, CallToAction: snapshot.CallToAction, ScheduledAt: timestamppb.New(snapshot.At),
			RunPosition: uint32(snapshot.RunPosition), RunSize: uint32(snapshot.RunSize), //nolint:gosec // 2 a run.
		},
	}
	if len(snapshot.Record) != 0 {
		record.Clip = &clipsv1.ClipRecord{}
		if err := protojson.Unmarshal(snapshot.Record, record.Clip); err != nil {
			return nil, fmt.Errorf("what the clip was is not a clips.v1.ClipRecord: %w", err)
		}
	}
	written, err := protojson.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to write what the post was: %w", err)
	}
	var document map[string]any
	if err := json.Unmarshal(written, &document); err != nil {
		return nil, fmt.Errorf("failed to read back what the post was: %w", err)
	}
	return document, nil
}

// The story a post told: false for a post with none, such as one made by hand.
func KeyOf(metadata map[string]any) (posts.StoryKey, bool) {
	flag, _ := metadata[flagKey].(string)
	continent, _ := metadata[continentKey].(string)
	angle, _ := metadata[angleKey].(string)
	round, _ := metadata[roundKey].(string)
	return posts.StoryKey{Flag: flag, Continent: continent, Angle: posts.Angle(angle), Round: round}, flag != "" && continent != ""
}

// The countries a post showed: what is not a country code is left out.
func CountriesOf(metadata map[string]any) []string {
	kept, _ := metadata[countriesKey].([]any)
	var countries []string
	for _, value := range kept {
		if country, ok := value.(string); ok && country != "" {
			countries = append(countries, country)
		}
	}
	return countries
}
