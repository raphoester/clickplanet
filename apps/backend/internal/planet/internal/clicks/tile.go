package clicks

import (
	"context"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Tile struct {
	id      uint32
	owner   string
	shields int
	holder  cpsession.AccountID
}

type TileSnapshot struct {
	ID      uint32
	Owner   string
	Shields int
	Holder  cpsession.AccountID
}

func (s TileSnapshot) Restore() Tile {
	return Tile{id: s.ID, owner: s.Owner, shields: s.Shields, holder: s.Holder}
}

func (t Tile) TakeSnapshot() TileSnapshot {
	return TileSnapshot{ID: t.id, Owner: t.owner, Shields: t.shields, Holder: t.holder}
}

func (t Tile) Claimed(flag string, by cpsession.AccountID) (Tile, Impact) {
	impact := Impact{Tile: t.id, Owner: t.owner, Outcome: OutcomeOf(t.owner, flag, t.shields)}
	switch impact.Outcome {
	case Taken:
		t = Tile{id: t.id, owner: flag, holder: by}
	case Shielded:
		t.shields--
	case Unchanged:
	}
	impact.Shields = t.shields

	return t, impact
}

func holderOf(ctx context.Context) cpsession.AccountID {
	account, err := uuid.Parse(cpctx.GetAccount(ctx))
	if err != nil {
		return cpsession.NoAccount
	}
	return cpsession.AccountID(account)
}
