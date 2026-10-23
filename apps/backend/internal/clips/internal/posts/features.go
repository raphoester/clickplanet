package posts

// What a clip was, as the generator chose it and the replay played it (`clips.v1.ClipFeatures`): kept with its post and
// never published, so a later step can set it beside how the post did. Nothing here chooses, words or times a post.
type Features struct {
	DurationSeconds float64
	// The country code of the anthem under the clip; empty when it plays none.
	Music             string
	MusicStartSeconds float64
	// "front" or "tour".
	Camera string
	// "dive" or "flags"; empty on a tour.
	Look              string
	StoryKind         string
	SpanMinutes       float64
	TilesChanged      int
	StoryTilesChanged int
	PlaceTiles        int
	FlagTilesStart    int
	FlagTilesEnd      int
	LandmassesFlipped int
	Bombs             int
	Spreads           int
	Enclosures        int
	ScreensCrossed    float64
	TourStops         int
	Language          string
	Solidity          float64
}

// A clip written before the generator kept them has none: every clip lasts.
func (f Features) Recorded() bool {
	return f.DurationSeconds > 0
}
