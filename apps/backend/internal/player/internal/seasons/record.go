package seasons

import (
	"cmp"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Number uint32

type Country string

type Season struct {
	Number Number
	Flag   Country
	Won    bool
}

type RecordSnapshot struct {
	Account players.AccountID
	Seasons []Season
}

func (s RecordSnapshot) Restore() Record {
	played := slices.Clone(s.Seasons)
	slices.SortStableFunc(played, func(a, b Season) int { return cmp.Compare(a.Number, b.Number) })
	played = slices.CompactFunc(played, func(a, b Season) bool { return a.Number == b.Number })
	return Record{account: s.Account, seasons: played}
}

type Record struct {
	account players.AccountID
	seasons []Season
}

func NewRecord(account players.AccountID) Record {
	return RecordSnapshot{Account: account}.Restore()
}

func (r Record) TakeSnapshot() RecordSnapshot {
	return RecordSnapshot{Account: r.account, Seasons: slices.Clone(r.seasons)}
}

func (r Record) With(season Season) Record {
	others := slices.DeleteFunc(slices.Clone(r.seasons), func(played Season) bool { return played.Number == season.Number })
	return RecordSnapshot{Account: r.account, Seasons: append(others, season)}.Restore()
}

func (r Record) Forgotten() Record {
	return NewRecord(r.account)
}

func (r Record) Played(number Number) bool {
	return slices.ContainsFunc(r.seasons, func(played Season) bool { return played.Number == number })
}

func (r Record) Won(number Number) bool {
	return slices.ContainsFunc(r.seasons, func(played Season) bool { return played.Number == number && played.Won })
}
