//go:build testing

package guard

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewInMemory(
	config Config,
	inForce *rules.InForce,
	clock cptime.Clock,
	observer Observer,
	scopeBans *bans.MemoryStore,
	accountBans *bans.MemoryStore,
	enforcer *bans.MemoryEnforcer,
	evidences *evidence.MemoryPersistence,
	heard ...evidence.Section,
) (*Guard, error) {
	book := bans.NewBook(inForce.Bans, config.Evidence.WithDefaults().Retention, clock,
		scopeBans, accountBans, cpatomic.FakeTransactioner{}, enforcer)

	return New(config, inForce, clock, observer, book, evidences, heard...)
}

func (g *Guard) EnforcedSentence(ctx context.Context, scope, account string) (Sentence, bool) {
	sentence, banned, err := g.banner.EnforcedSentence(ctx, bans.Caller{Scope: scope, Account: account})
	if err != nil {
		g.onStateError(err)
		return Sentence{}, false
	}

	return sentence, banned
}

func (g *Guard) Run(ctx context.Context) {
	g.Hearing().Run(ctx)
}
