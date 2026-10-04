package titles

import (
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type ID string

type Title interface {
	ID() ID
	Name() string
	EarnedBy(career Career) bool
}

type IDs []ID

func (i IDs) Without(held IDs) IDs {
	return slices.DeleteFunc(slices.Clone(i), func(id ID) bool { return slices.Contains(held, id) })
}

type Holdings map[players.AccountID]IDs

func (h Holdings) Len() int {
	total := 0
	for _, ids := range h {
		total += len(ids)
	}
	return total
}

type Reconciliation struct {
	grants      Holdings
	revocations Holdings
}

func (r Reconciliation) Grants() Holdings { return r.grants }

func (r Reconciliation) Revocations() Holdings { return r.revocations }

type Place struct {
	track     TrackID
	trackName string
	number    int
	count     int
}

func (p Place) Track() TrackID { return p.track }

func (p Place) TrackName() string { return p.trackName }

func (p Place) Number() int { return p.number }

func (p Place) Count() int { return p.count }

func (p Place) Ranked() bool { return p.count > 0 }

type Standing struct {
	title Title
	place Place
}

func (s Standing) Title() Title { return s.title }

func (s Standing) Place() Place { return s.place }

func (s Standing) Empty() bool { return s.title == nil }
