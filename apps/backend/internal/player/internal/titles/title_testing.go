//go:build testing

package titles

import "strings"

type FakeTitle struct {
	Key   ID
	Tiles uint64
}

func (f FakeTitle) ID() ID { return f.Key }

func (f FakeTitle) Name() string { return strings.ToUpper(string(f.Key)) }

func (f FakeTitle) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= f.Tiles }
