package progressionmessage

import (
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
)

func Title(standing titles.Standing) *progressionv1.Title {
	if standing.Empty() {
		return nil
	}

	title := &progressionv1.Title{Id: string(standing.Title().ID()), Name: standing.Title().Name(), Feat: standing.Feat()}
	if place := standing.Place(); place.Ranked() {
		title.Rank = &progressionv1.Rank{
			TrackId:   string(place.Track()),
			TrackName: place.TrackName(),
			Number:    uint32(place.Number()), //nolint:gosec // a place in a track of a handful of ranks.
			Count:     uint32(place.Count()),  //nolint:gosec // as above.
		}
	}
	return title
}

func Titles(standings []titles.Standing) []*progressionv1.Title {
	encoded := make([]*progressionv1.Title, 0, len(standings))
	for _, standing := range standings {
		encoded = append(encoded, Title(standing))
	}
	return encoded
}
