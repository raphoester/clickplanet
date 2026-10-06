package antibot_place_defender_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_defender_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_defender_usecase/antibot_place_defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct{ in place_defender_usecase.In }

func (s *stubUseCase) Execute(_ context.Context, in place_defender_usecase.In) (bonuses.Held, error) {
	s.in = in
	return bonuses.Held{}, nil
}

type bans struct{ scopes, accounts *cpcolls.Set[string] }

func (b bans) Banned(scope, account string) bool {
	return b.scopes.Contains(scope) || b.accounts.Contains(account)
}

func TestABannedCallersDefenderIsADud(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_place_defender.New(inner, bans{scopes: cpcolls.NewSet("2001:db8::/64"), accounts: cpcolls.NewSet[string]()})

	_, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), place_defender_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.True(t, inner.in.Dud)
}

func TestABannedAccountsDefenderIsADudFromAnyScope(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_place_defender.New(inner, bans{scopes: cpcolls.NewSet[string](), accounts: cpcolls.NewSet("a-guest")})

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), "a-guest")
	_, err := decorator.Execute(ctx, place_defender_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.True(t, inner.in.Dud)
}

func TestAnyoneElsesDefenderIsReal(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_place_defender.New(inner, bans{scopes: cpcolls.NewSet("2001:db8::/64"), accounts: cpcolls.NewSet[string]()})

	_, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), place_defender_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.False(t, inner.in.Dud)
}
