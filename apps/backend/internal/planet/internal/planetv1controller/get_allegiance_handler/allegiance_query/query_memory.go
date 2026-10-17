package allegiance_query

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
)

type Allegiances interface {
	Snapshot(account allegiances.Account) allegiances.Snapshot
}

func NewMemoryQuery(allegiances Allegiances) *MemoryQuery {
	return &MemoryQuery{allegiances: allegiances}
}

type MemoryQuery struct {
	allegiances Allegiances
}

func (q *MemoryQuery) Allegiance(account allegiances.Account) *planetv1.GetAllegianceResponse {
	if account == allegiances.NoAccount {
		return &planetv1.GetAllegianceResponse{}
	}

	kept := q.allegiances.Snapshot(account)
	if kept.Flag == "" {
		return &planetv1.GetAllegianceResponse{}
	}

	return &planetv1.GetAllegianceResponse{
		Allegiance: allegiancemessage.Encode(kept.Flag, allegiances.ChangedAt(kept.ChangedAt).ChangeableAt()),
	}
}
