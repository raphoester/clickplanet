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
	Grants      Holdings
	Revocations Holdings
}

type Place struct {
	Track     TrackID
	TrackName string
	Number    int
	Count     int
}

func (p Place) Ranked() bool { return p.Count > 0 }

type Standing struct {
	Title Title
	Place Place
}

func (s Standing) Empty() bool { return s.Title == nil }
