//go:build testing

package titles

import "strings"

type FakeTitle struct {
	Key   ID
	Tiles uint64
}

func (f FakeTitle) ID() ID { return f.Key }

func (f FakeTitle) Name() string { return strings.ToUpper(string(f.Key)) }

func (f FakeTitle) Threshold() uint64 { return f.Tiles }

func (f FakeTitle) EarnedBy(career Career) bool { return career.Stats().TilesTaken() >= f.Tiles }

type FakeTrack struct {
	Key   TrackID
	Steps []Rank
}

func (f FakeTrack) ID() TrackID { return f.Key }

func (f FakeTrack) Name() string { return strings.ToUpper(string(f.Key)) }

func (f FakeTrack) Ranks() []Rank { return f.Steps }

func (f FakeTrack) Progress(career Career) uint64 { return career.Stats().TilesTaken() }
