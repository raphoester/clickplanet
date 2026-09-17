package antibot_drop_bomb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/antibot_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct{ in drop_bomb_usecase.In }

func (s *stubUseCase) Execute(_ context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	s.in = in
	return clicks.Blast{}, nil
}

type bans struct{ scopes *cpcolls.Set[string] }

func (b bans) Banned(scope string) bool { return b.scopes.Contains(scope) }

func TestABannedCallersBombIsADud(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_drop_bomb.New(inner, bans{scopes: cpcolls.NewSet("2001:db8::/64")})

	_, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), drop_bomb_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.True(t, inner.in.Dud, "the ban is on the scope, not the address")
}

func TestAnyoneElsesBombIsReal(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_drop_bomb.New(inner, bans{scopes: cpcolls.NewSet("2001:db8::/64")})

	_, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), drop_bomb_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.False(t, inner.in.Dud)
}
