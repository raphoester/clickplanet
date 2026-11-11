//go:build testing

package guard

import (
	"context"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewInMemory(
	config Config,
	inForce *cprules.InForce[*antibotv1.Rules],
	clock cptime.Clock,
	observer Observer,
	scopeBans *bans.MemoryStore,
	accountBans *bans.MemoryStore,
	enforcer *bans.MemoryEnforcer,
	warned *warnings.MemoryStore,
	feed *warnings.MemoryFeed,
	evidences *evidence.MemoryPersistence,
	heard ...evidence.Section,
) *Guard {
	book := bans.NewBook(cprules.Reader(inForce, bans.Part), config.Evidence.WithDefaults().Retention, clock,
		scopeBans, accountBans, cpatomic.FakeTransactioner{}, enforcer)
	warner := warnings.NewBook(book, clock, warned, feed)

	return New(config, inForce, clock, observer, book, warner, evidences, heard...)
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
