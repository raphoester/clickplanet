package missions

import "time"

// What an account did that a mission may count. A fact may come twice: each kind keeps its own mark of what counted.
type Fact interface {
	counted(marks Counted) (Counted, bool)
}

type Counted struct {
	TakesBelow     uint64
	BombsBelow     uint64
	ShieldsBelow   uint64
	FortifiesBelow uint64
	BoxesUntil     time.Time
	AnswersUntil   time.Time
}

type Take struct {
	Flag     string
	Previous string
	Ground   string
	Position uint64
}

func (t Take) counted(marks Counted) (Counted, bool) {
	var fresh bool
	marks.TakesBelow, fresh = below(t.Position, marks.TakesBelow)
	return marks, fresh
}

// Ground is empty in the sea.
type BombLanded struct {
	Ground   string
	Position uint64
}

func (b BombLanded) counted(marks Counted) (Counted, bool) {
	var fresh bool
	marks.BombsBelow, fresh = below(b.Position, marks.BombsBelow)
	return marks, fresh
}

// Shields is what the tile held after it.
type ShieldPlaced struct {
	Shields  uint32
	Position uint64
}

func (s ShieldPlaced) counted(marks Counted) (Counted, bool) {
	var fresh bool
	marks.ShieldsBelow, fresh = below(s.Position, marks.ShieldsBelow)
	return marks, fresh
}

type Fortified struct {
	Tiles    uint32
	Position uint64
}

func (f Fortified) counted(marks Counted) (Counted, bool) {
	var fresh bool
	marks.FortifiesBelow, fresh = below(f.Position, marks.FortifiesBelow)
	return marks, fresh
}

type BoxCaught struct {
	At time.Time
}

func (b BoxCaught) counted(marks Counted) (Counted, bool) {
	var fresh bool
	marks.BoxesUntil, fresh = until(b.At, marks.BoxesUntil)
	return marks, fresh
}

type RightAnswer struct {
	At time.Time
}

func (r RightAnswer) counted(marks Counted) (Counted, bool) {
	var fresh bool
	marks.AnswersUntil, fresh = until(r.At, marks.AnswersUntil)
	return marks, fresh
}

func below(position, mark uint64) (uint64, bool) {
	if position < mark {
		return mark, false
	}
	return position + 1, true
}

// Postgres keeps a time to the microsecond: a fact told again must compare equal to the one kept.
func until(at, mark time.Time) (time.Time, bool) {
	moment := at.UTC().Truncate(time.Microsecond)
	if !moment.After(mark) {
		return mark, false
	}
	return moment, true
}
