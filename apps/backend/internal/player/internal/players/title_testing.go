//go:build testing

package players

import "strings"

type FakeTitle struct {
	Key   TitleID
	Tiles uint64
}

func (f FakeTitle) ID() TitleID { return f.Key }

func (f FakeTitle) Name() string { return strings.ToUpper(string(f.Key)) }

func (f FakeTitle) EarnedBy(stats Stats) bool { return stats.TilesTaken >= f.Tiles }
