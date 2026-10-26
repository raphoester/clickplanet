package inmemory_tile_storage

import (
	"context"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const noHolder = uint32(0)

// The accounts whose takes hold tiles, interned to four bytes, so a tile's holder is an index and not a uuid.
type holderbook struct {
	accounts []cpsession.AccountID
	ids      map[cpsession.AccountID]uint32
}

func newHolderbook() *holderbook {
	return &holderbook{
		accounts: []cpsession.AccountID{cpsession.NoAccount},
		ids:      map[cpsession.AccountID]uint32{cpsession.NoAccount: noHolder},
	}
}

func (h *holderbook) intern(account cpsession.AccountID) uint32 {
	if id, ok := h.ids[account]; ok {
		return id
	}

	id := uint32(len(h.accounts)) //nolint:gosec // one entry per account that ever held a tile since the boot.
	h.accounts = append(h.accounts, account)
	h.ids[account] = id

	return id
}

func (h *holderbook) idOf(account cpsession.AccountID) (uint32, bool) {
	id, ok := h.ids[account]
	return id, ok && id != noHolder
}

func (h *holderbook) accountOf(id uint32) cpsession.AccountID {
	return h.accounts[id]
}

// The slot stays, so no other id moves; it names nobody any more.
func (h *holderbook) forget(account cpsession.AccountID) {
	id, ok := h.idOf(account)
	if !ok {
		return
	}
	delete(h.ids, account)
	h.accounts[id] = cpsession.NoAccount
}

// The account a player's take is made for, as the session interceptor put it in the context.
func holderOf(ctx context.Context) cpsession.AccountID {
	account, err := uuid.Parse(cpctx.GetAccount(ctx))
	if err != nil {
		return cpsession.NoAccount
	}
	return cpsession.AccountID(account)
}
