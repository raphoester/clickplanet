package claude_editor

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var ErrNoSuchStory = errors.New("the findings hold no such story")

// What the editor was told of one story of the findings, alone, as JSON: its front, tour, place or saga in the shape
// the editor read it, the run's names, and for a saga the paths of its flags. A writer tells the story from it
// (docs/video-lab.md, "The war report's bench"). A seasonal story has none.
func FactsOf(findings posts.Findings, story posts.Story) (string, error) {
	written := struct {
		Now   time.Time         `json:"now"`
		Names map[string]string `json:"names"`
		Front *front            `json:"front,omitempty"`
		Tour  *tour             `json:"tour,omitempty"`
		Place *place            `json:"place,omitempty"`
		Saga  *saga             `json:"saga,omitempty"`
		Paths *paths            `json:"paths,omitempty"`
	}{Now: findings.Until.UTC().Truncate(time.Second), Names: findings.Names}

	switch {
	case story.Front > 0 && story.Front <= len(findings.Fronts):
		found := frontOf(story.Front, findings.Fronts[story.Front-1])
		written.Front = &found
	case story.Tour > 0 && story.Tour <= len(findings.Tours):
		found := tourOf(story.Tour, findings.Tours[story.Tour-1])
		written.Tour = &found
	case story.Place > 0 && story.Place <= len(findings.Places):
		found := placeOf(story.Place, findings.Places[story.Place-1])
		written.Place = &found
	case story.Saga > 0 && story.Saga <= len(findings.Sagas):
		found := sagaOf(story.Saga, findings.Sagas[story.Saga-1])
		written.Saga = &found
		sides := posts.Findings{Paths: slices.DeleteFunc(slices.Clone(findings.Paths), func(path posts.Path) bool {
			return path.Flag != story.Attacker && path.Flag != story.Rival
		}), ContinentTiles: findings.ContinentTiles}
		if len(sides.Paths) > 0 {
			kept := pathsOf(sides)
			written.Paths = &kept
		}
	default:
		return "", fmt.Errorf("%w: %s", ErrNoSuchStory, story.Names())
	}

	text, err := json.Marshal(written)
	if err != nil {
		return "", fmt.Errorf("failed to write the story's facts: %w", err)
	}
	return string(text), nil
}
