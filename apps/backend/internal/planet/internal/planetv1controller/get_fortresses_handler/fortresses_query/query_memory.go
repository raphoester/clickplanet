package fortresses_query

import (
	"cmp"
	"slices"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Fortresses interface {
	Fortresses() map[clicks.ProvinceID]string
}

func NewMemoryQuery(fortresses Fortresses) *MemoryQuery {
	return &MemoryQuery{fortresses: fortresses}
}

type MemoryQuery struct {
	fortresses Fortresses
}

func (q *MemoryQuery) Fortresses() *planetv1.GetFortressesResponse {
	held := q.fortresses.Fortresses()

	fortresses := make([]*planetv1.Fortress, 0, len(held))
	for province, country := range held {
		fortresses = append(fortresses, &planetv1.Fortress{ProvinceId: uint32(province), CountryId: country})
	}
	slices.SortFunc(fortresses, func(a, b *planetv1.Fortress) int {
		return cmp.Compare(a.GetProvinceId(), b.GetProvinceId())
	})

	return &planetv1.GetFortressesResponse{Fortresses: fortresses}
}
